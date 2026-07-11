package codexsdk

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ronhuafeng/codexsdk-go/codexsdk/protocolv2"
)

func TestExactRunnerStartPreservesGeneratedFacts(t *testing.T) {
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("happy")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	input := []protocolv2.UserInput{protocolv2.NewUserInputText(protocolv2.UserInputText{Text: "hello"})}
	result, err := root.ThreadRunner().Start(context.Background(), StartThreadRunRequest{
		Thread: protocolv2.ThreadStartParams{Model: protocolv2.Value("gpt-exact")},
		Turn:   protocolv2.TurnStartParams{Input: input},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Start.Thread.ID == "" || result.Run.Turn.ID == "" || result.Run.Turn.Status != protocolv2.TurnStatusCompleted {
		t.Fatalf("result did not preserve start/turn facts: %#v", result)
	}
	if result.Run.FinalResponse != "final-"+result.Run.Turn.ID {
		t.Fatalf("final response = %q", result.Run.FinalResponse)
	}
	if result.Run.Usage == nil || result.Run.Usage.Total.InputTokens != 30 {
		t.Fatalf("usage = %#v", result.Run.Usage)
	}
	if len(result.Run.Notifications) != 3 {
		t.Fatalf("notifications = %d, want 3", len(result.Run.Notifications))
	}
	if result.Run.InputStats.ItemsCount != 1 || result.Run.InputStats.TextBytes != 5 || result.Run.InputStats.InputItemsHash == "" {
		t.Fatalf("input stats = %#v", result.Run.InputStats)
	}
}

func TestExactRunnerPassesThroughGeneratedParamsAndOwnsOnlyThreadID(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("happy")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	thread := protocolv2.ThreadStartParams{
		CWD:       protocolv2.Value("/exact/cwd"),
		Ephemeral: protocolv2.Value(true),
		Model:     protocolv2.Value("gpt-exact"),
		Sandbox:   protocolv2.Value(protocolv2.SandboxModeReadOnly),
	}
	turn := protocolv2.TurnStartParams{
		CWD:    protocolv2.Value("/turn/cwd"),
		Effort: protocolv2.Value(protocolv2.ReasoningEffort("high")),
		Input: []protocolv2.UserInput{
			protocolv2.NewUserInputText(protocolv2.UserInputText{Text: "hello"}),
		},
	}
	request := StartThreadRunRequest{Thread: thread, Turn: turn}
	if _, err := root.ThreadRunner().Start(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if request.Turn.ThreadID != "" {
		t.Fatalf("runner mutated caller request: %#v", request.Turn)
	}
	records := readRecords(t, record)
	startParams := firstRecord(records, "recv", protocolv2.MethodThreadStart)["params"].(map[string]any)
	if startParams["cwd"] != "/exact/cwd" || startParams["model"] != "gpt-exact" || startParams["ephemeral"] != true || startParams["sandbox"] != "read-only" {
		t.Fatalf("thread/start params = %#v", startParams)
	}
	turnParams := firstRecord(records, "recv", protocolv2.MethodTurnStart)["params"].(map[string]any)
	if turnParams["cwd"] != "/turn/cwd" || turnParams["effort"] != "high" || turnParams["threadId"] == "" {
		t.Fatalf("turn/start params = %#v", turnParams)
	}
}

func TestExactRunnerResumePreservesResponseAndComposesTurn(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("happy")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	result, err := root.ThreadRunner().Resume(context.Background(), ResumeThreadRunRequest{
		Thread: protocolv2.ThreadResumeParams{ThreadID: "thread-resume", Model: protocolv2.Value("gpt-resume")},
		Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{
			protocolv2.NewUserInputText(protocolv2.UserInputText{Text: "resume"}),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Resume.Thread.ID != "thread-resume" || result.Run.Turn.Status != protocolv2.TurnStatusCompleted {
		t.Fatalf("resume result = %#v", result)
	}
	turnParams := firstRecord(readRecords(t, record), "recv", protocolv2.MethodTurnStart)["params"].(map[string]any)
	if turnParams["threadId"] != "thread-resume" {
		t.Fatalf("turn/start params = %#v", turnParams)
	}
}

func TestNotificationHandlerReceivesExactNotificationsInOrder(t *testing.T) {
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	var mu sync.Mutex
	var kinds []protocolv2.ServerNotificationKind
	root, err := New(ClientOptions{
		CWD:     t.TempDir(),
		Command: fakeCommand("happy"),
		ServerNotificationHandler: func(_ context.Context, notification protocolv2.ServerNotification) error {
			mu.Lock()
			kinds = append(kinds, notification.Kind())
			mu.Unlock()
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = root.ThreadRunner().Start(context.Background(), StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := root.Close(); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	want := []protocolv2.ServerNotificationKind{
		protocolv2.ServerNotificationKindItemCompleted,
		protocolv2.ServerNotificationKindThreadTokenUsageUpdated,
		protocolv2.ServerNotificationKindTurnCompleted,
	}
	if !reflect.DeepEqual(kinds, want) {
		t.Fatalf("handler order = %#v, want %#v", kinds, want)
	}
}

func TestNotificationHandlerFailureCancelsStreamWithPartialEvidence(t *testing.T) {
	handlerErr := errors.New("handler failed")
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{
		CWD:     t.TempDir(),
		Command: fakeCommand("happy"),
		ServerNotificationHandler: func(context.Context, protocolv2.ServerNotification) error {
			return handlerErr
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, runErr := root.ThreadRunner().Start(context.Background(), StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}})
	if !errors.Is(runErr, ErrHandlerFailed) || !errors.Is(runErr, handlerErr) {
		t.Fatalf("run error = %v, want handler cause", runErr)
	}
	if len(result.Run.Notifications) == 0 {
		t.Fatal("handler failure erased accepted run notification")
	}
	if closeErr := root.Close(); !errors.Is(closeErr, handlerErr) {
		t.Fatalf("Close error = %v, want first handler cause", closeErr)
	}
}

func TestNotificationHandlerPanicBecomesTypedClientFailure(t *testing.T) {
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{
		CWD:     t.TempDir(),
		Command: fakeCommand("happy"),
		ServerNotificationHandler: func(context.Context, protocolv2.ServerNotification) error {
			panic("boom")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = root.ThreadRunner().Start(context.Background(), StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}})
	if closeErr := root.Close(); !errors.Is(closeErr, ErrHandlerFailed) {
		t.Fatalf("Close error = %v, want ErrHandlerFailed", closeErr)
	}
}

func TestNormalCloseWaitsForAcceptedNotificationHandler(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{
		CWD:     t.TempDir(),
		Command: fakeCommand("happy"),
		ServerNotificationHandler: func(context.Context, protocolv2.ServerNotification) error {
			select {
			case <-started:
			default:
				close(started)
			}
			<-release
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := root.ThreadRunner().StartStream(context.Background(), StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}})
	if err != nil {
		t.Fatal(err)
	}
	for stream.Next(context.Background()) {
	}
	<-started
	closed := make(chan error, 1)
	go func() { closed <- root.Close() }()
	select {
	case err := <-closed:
		t.Fatalf("Close returned before handler: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	if err := <-closed; err != nil {
		t.Fatal(err)
	}
}

func TestNotificationQueueOverflowPreservesFirstFailure(t *testing.T) {
	started := make(chan struct{})
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{
		CWD:                       t.TempDir(),
		Command:                   fakeCommand("happy"),
		NotificationQueueCapacity: 1,
		ServerNotificationHandler: func(ctx context.Context, _ protocolv2.ServerNotification) error {
			select {
			case <-started:
			default:
				close(started)
			}
			<-ctx.Done()
			return ctx.Err()
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = root.ThreadRunner().Start(context.Background(), StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}})
	<-started
	if closeErr := root.Close(); !errors.Is(closeErr, ErrNotificationBackpressure) {
		t.Fatalf("Close error = %v, want ErrNotificationBackpressure", closeErr)
	}
}

func TestExactServerRequestHandlerUsesGeneratedRequestAndTypedResponse(t *testing.T) {
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	seen := make(chan protocolv2.ServerRequestKind, 1)
	root, err := New(ClientOptions{
		CWD:     t.TempDir(),
		Command: fakeCommand("approval"),
		ServerRequestHandler: func(_ context.Context, request protocolv2.ServerRequest) (ServerRequestResponse, error) {
			seen <- request.Kind()
			return CommandExecutionApprovalResponse(protocolv2.CommandExecutionRequestApprovalResponse{
				Decision: protocolv2.NewCommandExecutionApprovalDecisionAccept(),
			}), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	result, err := root.ThreadRunner().Start(context.Background(), StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Run.FinalResponse == "" {
		t.Fatal("run did not complete after exact approval response")
	}
	if got := <-seen; got != protocolv2.ServerRequestKindItemCommandExecutionRequestApproval {
		t.Fatalf("request kind = %s", got)
	}
}

func TestExactServerRequestHandlerRejectsMismatchedAndEmptyResponses(t *testing.T) {
	for _, test := range []struct {
		name     string
		response ServerRequestResponse
	}{
		{name: "mismatched", response: FileChangeApprovalResponse(protocolv2.FileChangeRequestApprovalResponse{Decision: protocolv2.FileChangeApprovalDecisionDecline})},
		{name: "empty", response: ServerRequestResponse{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
			root, err := New(ClientOptions{
				CWD:     t.TempDir(),
				Command: fakeCommand("approval"),
				ServerRequestHandler: func(context.Context, protocolv2.ServerRequest) (ServerRequestResponse, error) {
					return test.response, nil
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			_, runErr := root.ThreadRunner().Start(context.Background(), StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}})
			if !errors.Is(runErr, ErrExactServerRequest) {
				t.Fatalf("run error = %v, want typed exact server request failure", runErr)
			}
			if closeErr := root.Close(); !errors.Is(closeErr, ErrExactServerRequest) {
				t.Fatalf("Close error = %v, want first typed exact server request failure", closeErr)
			}
		})
	}
}

func TestExactRunWithoutHandlerFailsClosedInsteadOfUsingLegacyPendingQueue(t *testing.T) {
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("approval")})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, runErr := root.ThreadRunner().Start(ctx, StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}})
	if runErr != nil {
		t.Fatalf("nil-handler command approval should decline and complete: %v (result %#v)", runErr, result)
	}
	client := root.(*client)
	client.turnMu.Lock()
	pending := len(client.pendingServer)
	client.turnMu.Unlock()
	if pending != 0 {
		t.Fatalf("exact request entered legacy pending queue: %#v", client.pendingServer)
	}
	if err := root.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestExactRunNilHandlerFailClosedResponsesAreDeterministic(t *testing.T) {
	for _, mode := range []string{"approval", "file-approval", "user-input", "approval-before-turn-start"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
			root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand(mode)})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if _, err := root.ThreadRunner().Start(ctx, StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}}); err != nil {
				t.Fatalf("exact nil-handler %s did not fail closed: %v", mode, err)
			}
			client := root.(*client)
			client.turnMu.Lock()
			pending := len(client.pendingServer)
			client.turnMu.Unlock()
			if pending != 0 {
				t.Fatalf("exact request entered legacy pending queue: %#v", client.pendingServer)
			}
			if err := root.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestExactRunNilHandlerUnsafeRequestPreservesPartialEvidenceAndTypedFirstCause(t *testing.T) {
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("auth-refresh-after-notification")})
	if err != nil {
		t.Fatal(err)
	}
	result, runErr := root.ThreadRunner().Start(context.Background(), StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}})
	if !errors.Is(runErr, ErrExactServerRequest) {
		t.Fatalf("run error = %v, want typed exact server request cause", runErr)
	}
	if len(result.Run.Notifications) == 0 {
		t.Fatal("exact fail-closed termination erased accepted notification evidence")
	}
	if closeErr := root.Close(); !errors.Is(closeErr, ErrExactServerRequest) {
		t.Fatalf("Close error = %v, want first exact server request cause", closeErr)
	}
	client := root.(*client)
	client.turnMu.Lock()
	pending := len(client.pendingServer)
	client.turnMu.Unlock()
	if pending != 0 {
		t.Fatalf("unsafe exact request entered legacy pending queue: %#v", client.pendingServer)
	}
}

func TestExactFailClosedCoverageMatchesGeneratedRequestKinds(t *testing.T) {
	requests := []protocolv2.ServerRequest{
		protocolv2.NewServerRequestItemCommandExecutionRequestApproval(protocolv2.ServerRequestItemCommandExecutionRequestApproval{}),
		protocolv2.NewServerRequestItemFileChangeRequestApproval(protocolv2.ServerRequestItemFileChangeRequestApproval{}),
		protocolv2.NewServerRequestItemToolRequestUserInput(protocolv2.ServerRequestItemToolRequestUserInput{}),
		protocolv2.NewServerRequestMCPServerElicitationRequest(protocolv2.ServerRequestMCPServerElicitationRequest{}),
		protocolv2.NewServerRequestItemPermissionsRequestApproval(protocolv2.ServerRequestItemPermissionsRequestApproval{}),
		protocolv2.NewServerRequestItemToolCall(protocolv2.ServerRequestItemToolCall{}),
		protocolv2.NewServerRequestAccountChatGPTAuthTokensRefresh(protocolv2.ServerRequestAccountChatGPTAuthTokensRefresh{}),
		protocolv2.NewServerRequestAttestationGenerate(protocolv2.ServerRequestAttestationGenerate{}),
		protocolv2.NewServerRequestCurrentTimeRead(protocolv2.ServerRequestCurrentTimeRead{}),
		protocolv2.NewServerRequestApplyPatchApproval(protocolv2.ServerRequestApplyPatchApproval{}),
		protocolv2.NewServerRequestExecCommandApproval(protocolv2.ServerRequestExecCommandApproval{}),
	}
	for _, request := range requests {
		response, synthesizable := exactFailClosedServerRequestResponse(request)
		if synthesizable != (response.kind == request.Kind() && response.value != nil) {
			t.Fatalf("fail-closed coverage for %s is inconsistent: %#v", request.Kind(), response)
		}
	}
}

func TestNormalCloseCancelsAndJoinsExactServerRequestHandler(t *testing.T) {
	started := make(chan struct{})
	finished := make(chan struct{})
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{
		CWD:     t.TempDir(),
		Command: fakeCommand("approval"),
		ServerRequestHandler: func(ctx context.Context, _ protocolv2.ServerRequest) (ServerRequestResponse, error) {
			close(started)
			<-ctx.Done()
			close(finished)
			return ServerRequestResponse{}, ctx.Err()
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := root.ThreadRunner().StartStream(context.Background(), StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}})
	if err != nil {
		t.Fatal(err)
	}
	<-started
	if err := root.Close(); err != nil {
		t.Fatalf("normal Close returned handler cancellation as failure: %v", err)
	}
	<-finished
	if !errors.Is(stream.Err(), ErrClientClosed) {
		t.Fatalf("stream error = %v, want ErrClientClosed", stream.Err())
	}
}

func TestServerRequestResponseConstructorsCoverGeneratedRequestKinds(t *testing.T) {
	responses := []ServerRequestResponse{
		CommandExecutionApprovalResponse(protocolv2.CommandExecutionRequestApprovalResponse{}),
		FileChangeApprovalResponse(protocolv2.FileChangeRequestApprovalResponse{}),
		ToolUserInputResponse(protocolv2.ToolRequestUserInputResponse{}),
		MCPElicitationResponse(protocolv2.McpServerElicitationRequestResponse{}),
		PermissionsApprovalResponse(protocolv2.PermissionsRequestApprovalResponse{}),
		DynamicToolResponse(protocolv2.DynamicToolCallResponse{}),
		ChatGPTAuthRefreshResponse(protocolv2.ChatgptAuthTokensRefreshResponse{}),
		AttestationResponse(protocolv2.AttestationGenerateResponse{}),
		CurrentTimeResponse(protocolv2.CurrentTimeReadResponse{}),
		ApplyPatchApprovalResponse(protocolv2.ApplyPatchApprovalResponse{}),
		ExecCommandApprovalResponse(protocolv2.ExecCommandApprovalResponse{}),
	}
	want := []protocolv2.ServerRequestKind{
		protocolv2.ServerRequestKindItemCommandExecutionRequestApproval,
		protocolv2.ServerRequestKindItemFileChangeRequestApproval,
		protocolv2.ServerRequestKindItemToolRequestUserInput,
		protocolv2.ServerRequestKindMCPServerElicitationRequest,
		protocolv2.ServerRequestKindItemPermissionsRequestApproval,
		protocolv2.ServerRequestKindItemToolCall,
		protocolv2.ServerRequestKindAccountChatGPTAuthTokensRefresh,
		protocolv2.ServerRequestKindAttestationGenerate,
		protocolv2.ServerRequestKindCurrentTimeRead,
		protocolv2.ServerRequestKindApplyPatchApproval,
		protocolv2.ServerRequestKindExecCommandApproval,
	}
	if len(responses) != len(want) {
		t.Fatalf("response constructors = %d, generated kinds = %d", len(responses), len(want))
	}
	for index := range want {
		if responses[index].kind != want[index] || responses[index].value == nil {
			t.Fatalf("constructor %d = %#v, want %s", index, responses[index], want[index])
		}
	}
}

func TestExactRunnerStreamOrdersNotificationsAndReturnsSnapshots(t *testing.T) {
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("happy")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	stream, err := root.ThreadRunner().StartStream(context.Background(), StartThreadRunRequest{
		Thread: protocolv2.ThreadStartParams{Model: protocolv2.Value("gpt-exact")},
		Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{
			protocolv2.NewUserInputText(protocolv2.UserInputText{Text: "hello"}),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var kinds []protocolv2.ServerNotificationKind
	for stream.Next(context.Background()) {
		kinds = append(kinds, stream.Notification().Kind())
	}
	want := []protocolv2.ServerNotificationKind{
		protocolv2.ServerNotificationKindItemCompleted,
		protocolv2.ServerNotificationKindThreadTokenUsageUpdated,
		protocolv2.ServerNotificationKindTurnCompleted,
	}
	if !reflect.DeepEqual(kinds, want) {
		t.Fatalf("notification order = %#v, want %#v", kinds, want)
	}
	first, ok := stream.Result()
	if !ok || stream.Err() != nil {
		t.Fatalf("result ok=%v err=%v", ok, stream.Err())
	}
	first.Run.Notifications = nil
	first.Run.Usage.Total.InputTokens = 999
	second, ok := stream.Result()
	if !ok || len(second.Run.Notifications) != 3 || second.Run.Usage.Total.InputTokens != 30 {
		t.Fatalf("result snapshot was mutable: %#v", second.Run)
	}
}

func TestExactRunnerRejectsOwnedThreadIDBeforeTransport(t *testing.T) {
	runner := &exactRunner{client: &client{}}
	_, err := runner.StartStream(context.Background(), StartThreadRunRequest{
		Turn: protocolv2.TurnStartParams{ThreadID: "caller-owned", Input: []protocolv2.UserInput{}},
	})
	if err == nil {
		t.Fatal("StartStream accepted caller-owned Turn.ThreadID")
	}
}

func TestExactRunnerPreservesPartialStartOnTurnStartFailure(t *testing.T) {
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("turn-start-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	stream, err := root.ThreadRunner().StartStream(context.Background(), StartThreadRunRequest{
		Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, ok := stream.Result()
	if !ok || result.Start.Thread.ID == "" {
		t.Fatalf("partial result = %#v, ok=%v", result, ok)
	}
	if stream.Err() == nil {
		t.Fatal("stream did not report post-start failure")
	}
}

func TestExactRunnerReturnsTypedTurnFailureWithPartialTurn(t *testing.T) {
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("failed")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	result, err := root.ThreadRunner().Start(context.Background(), StartThreadRunRequest{
		Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}},
	})
	var turnErr *TurnError
	if !errors.Is(err, ErrTurnFailed) || !errors.As(err, &turnErr) {
		t.Fatalf("error = %v, want TurnError/ErrTurnFailed", err)
	}
	if result.Start.Thread.ID == "" || result.Run.Turn.Status != protocolv2.TurnStatusFailed {
		t.Fatalf("partial result = %#v", result)
	}
}

func TestExactRunnerProducesSanitizedDiagnosticForMalformedNotification(t *testing.T) {
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("malformed-notification")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	result, err := root.ThreadRunner().Start(context.Background(), StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}})
	if err == nil {
		t.Fatal("run accepted malformed notification")
	}
	if len(result.Run.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v", result.Run.Diagnostics)
	}
	diagnostic := result.Run.Diagnostics[0]
	if diagnostic.Kind != "notification_decode_error" || diagnostic.Path != protocolv2.MethodItemAgentMessageDelta || diagnostic.SHA256 == "" || diagnostic.SizeBytes == 0 {
		t.Fatalf("diagnostic = %#v", diagnostic)
	}
}
