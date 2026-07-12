package codexsdk

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
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
	handlerEntered := make(chan struct{})
	releaseFailure := make(chan struct{})
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{
		CWD:     t.TempDir(),
		Command: fakeCommand("happy"),
		ServerNotificationHandler: func(_ context.Context, notification protocolv2.ServerNotification) error {
			if notification.Kind() != protocolv2.ServerNotificationKindTurnCompleted {
				return nil
			}
			close(handlerEntered)
			<-releaseFailure
			return handlerErr
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	pendingSeen := make(chan struct{})
	releaseAttach := make(chan struct{})
	client := root.(*client)
	client.testBeforeExactTurnAttach = func() { <-releaseAttach }
	client.testPendingExactNotification = func(notification rpcNotification) {
		if notification.method == protocolv2.MethodTurnCompleted {
			close(pendingSeen)
		}
	}
	type outcome struct {
		result StartedThreadRun
		err    error
	}
	finished := make(chan outcome, 1)
	go func() {
		result, runErr := root.ThreadRunner().Start(context.Background(), StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}})
		finished <- outcome{result: result, err: runErr}
	}()
	<-pendingSeen
	select {
	case <-handlerEntered:
		t.Fatal("global handler ran before pending exact evidence was appended")
	default:
	}
	close(releaseAttach)
	<-handlerEntered
	select {
	case got := <-finished:
		t.Fatalf("run completed before terminal notification handler published its result: %v", got.err)
	default:
	}
	close(releaseFailure)
	got := <-finished
	result, runErr := got.result, got.err
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

func TestNotificationDispatchFenceCompletesOnCancellationAndShutdownDrain(t *testing.T) {
	notification := protocolv2.NewServerNotificationConfigWarning(protocolv2.ServerNotificationConfigWarning{
		Params: protocolv2.ConfigWarningNotification{Summary: "fenced"},
	})

	t.Run("cancellation discards accepted fence", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		fence := make(chan struct{})
		c := &client{
			ctx:            ctx,
			dispatcherDone: make(chan struct{}),
			dispatchStop:   make(chan struct{}),
			notifications:  make(chan acceptedNotification, 1),
			options: ClientOptions{ServerNotificationHandler: func(context.Context, protocolv2.ServerNotification) error {
				t.Fatal("cancelled dispatcher invoked handler")
				return nil
			}},
		}
		c.handlerWG.Add(1)
		c.notifications <- acceptedNotification{notification: notification, evidenceReady: make(chan struct{}), dispatched: fence}
		go c.notificationDispatcher()
		<-c.dispatcherDone
		select {
		case <-fence:
		default:
			t.Fatal("cancelled dispatcher left accepted notification fence incomplete")
		}
		c.handlerWG.Wait()
	})

	t.Run("normal shutdown drains accepted fence", func(t *testing.T) {
		fence := make(chan struct{})
		c := &client{
			ctx:            context.Background(),
			dispatcherDone: make(chan struct{}),
			dispatchStop:   make(chan struct{}),
			notifications:  make(chan acceptedNotification, 1),
			options: ClientOptions{ServerNotificationHandler: func(context.Context, protocolv2.ServerNotification) error {
				return nil
			}},
		}
		c.handlerWG.Add(1)
		c.notifications <- acceptedNotification{notification: notification, evidenceReady: make(chan struct{}), dispatched: fence}
		close(c.dispatchStop)
		go c.notificationDispatcher()
		<-c.dispatcherDone
		select {
		case <-fence:
		default:
			t.Fatal("shutdown drain left accepted notification fence incomplete")
		}
		c.handlerWG.Wait()
	})
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
	for stream.Next(context.Background()) {
	}
}

func TestNotificationQueueOverflowPreservesFirstFailure(t *testing.T) {
	started := make(chan struct{})
	notificationAccepted := filepath.Join(t.TempDir(), "notification-accepted")
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{
		CWD:                       t.TempDir(),
		Command:                   fakeNotificationOverflowCommand(notificationAccepted),
		NotificationQueueCapacity: 1,
		ServerNotificationHandler: func(ctx context.Context, _ protocolv2.ServerNotification) error {
			if err := os.WriteFile(notificationAccepted, []byte("accepted"), 0o600); err != nil {
				return err
			}
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
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("notification handler did not accept the first notification")
	}
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
	notificationAccepted := filepath.Join(t.TempDir(), "notification-accepted")
	root, err := New(ClientOptions{
		CWD:     t.TempDir(),
		Command: fakeAuthRefreshAfterNotificationCommand(notificationAccepted),
		ServerNotificationHandler: func(context.Context, protocolv2.ServerNotification) error {
			return os.WriteFile(notificationAccepted, []byte("accepted"), 0o600)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, runErr := root.ThreadRunner().Start(ctx, StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}})
	if !errors.Is(runErr, ErrExactServerRequest) {
		t.Fatalf("run error = %v, want typed exact server request cause", runErr)
	}
	if errors.Is(runErr, context.DeadlineExceeded) {
		t.Fatalf("unsafe exact request was retained instead of rejected: %v", runErr)
	}
	if len(result.Run.Notifications) == 0 {
		t.Fatal("exact fail-closed termination erased accepted notification evidence")
	}
	if closeErr := root.Close(); !errors.Is(closeErr, ErrExactServerRequest) {
		t.Fatalf("Close error = %v, want first exact server request cause", closeErr)
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
	const callers = 8
	closed := make(chan error, callers)
	for range callers {
		go func() { closed <- root.Close() }()
	}
	for range callers {
		if err := <-closed; err != nil {
			t.Fatalf("concurrent normal Close returned handler cancellation as failure: %v", err)
		}
	}
	<-finished
	if !errors.Is(stream.Err(), ErrClientClosed) {
		t.Fatalf("stream error = %v, want ErrClientClosed", stream.Err())
	}
}

func TestCloseJoinsNotificationAndServerRequestHandlersAcceptedBeforeBoundary(t *testing.T) {
	notificationStarted := make(chan struct{})
	requestStarted := make(chan struct{})
	requestFinished := make(chan struct{})
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{
		CWD:     t.TempDir(),
		Command: fakeCommand("notification-and-approval"),
		ServerNotificationHandler: func(context.Context, protocolv2.ServerNotification) error {
			close(notificationStarted)
			<-requestFinished
			return nil
		},
		ServerRequestHandler: func(ctx context.Context, _ protocolv2.ServerRequest) (ServerRequestResponse, error) {
			close(requestStarted)
			<-ctx.Done()
			close(requestFinished)
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
	<-notificationStarted
	<-requestStarted
	closed := make(chan error, 1)
	go func() { closed <- root.Close() }()
	if err := <-closed; err != nil {
		t.Fatalf("normal Close returned handler cancellation as failure: %v", err)
	}
	<-requestFinished
	if !errors.Is(stream.Err(), ErrClientClosed) {
		t.Fatalf("stream error = %v, want ErrClientClosed", stream.Err())
	}
}

func TestRequestArrivingDuringCloseFailsClosedWithoutStartingNewHandler(t *testing.T) {
	record := tempRecord(t)
	sentinel := t.TempDir() + "/admission-closed"
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondCalled := make(chan struct{}, 1)
	var calls atomic.Int32
	root, err := New(ClientOptions{
		CWD:     t.TempDir(),
		Command: fakeCommand("late-approval-during-close", sentinel),
		ServerRequestHandler: func(ctx context.Context, _ protocolv2.ServerRequest) (ServerRequestResponse, error) {
			if calls.Add(1) != 1 {
				secondCalled <- struct{}{}
				return ServerRequestResponse{}, errors.New("late handler invoked")
			}
			close(firstStarted)
			<-ctx.Done()
			if err := os.WriteFile(sentinel, []byte("closed"), 0o600); err != nil {
				return ServerRequestResponse{}, err
			}
			<-releaseFirst
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
	<-firstStarted
	closed := make(chan error, 1)
	go func() { closed <- root.Close() }()
	if !waitForRecord(t, record, "recv-response", "", time.Second) {
		t.Fatal("late exact request did not receive a fail-closed response")
	}
	select {
	case <-secondCalled:
		t.Fatal("exact handler started for a request arriving after Close closed admission")
	default:
	}
	close(releaseFirst)
	if err := <-closed; err != nil {
		t.Fatalf("normal Close returned handler cancellation as failure: %v", err)
	}
	if !errors.Is(stream.Err(), ErrClientClosed) {
		t.Fatalf("stream error = %v, want ErrClientClosed", stream.Err())
	}
}

func TestFailureShutdownRejectsLateRequestWithoutStartingHandler(t *testing.T) {
	record := tempRecord(t)
	notificationAccepted := filepath.Join(t.TempDir(), "notification-accepted")
	release := filepath.Join(t.TempDir(), "failure-observed")
	lateSent := filepath.Join(t.TempDir(), "late-request-sent")
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	firstStarted := make(chan struct{})
	notificationStarted := make(chan struct{})
	allowNotificationFinish := make(chan struct{})
	lateCalled := make(chan struct{}, 1)
	var calls atomic.Int32
	root, err := New(ClientOptions{
		CWD:     t.TempDir(),
		Command: fakeLateApprovalDuringFailureCommand(notificationAccepted, release, lateSent),
		ServerNotificationHandler: func(ctx context.Context, _ protocolv2.ServerNotification) error {
			close(notificationStarted)
			if err := os.WriteFile(notificationAccepted, []byte("accepted"), 0o600); err != nil {
				return err
			}
			<-ctx.Done()
			<-allowNotificationFinish
			return ctx.Err()
		},
		ServerRequestHandler: func(context.Context, protocolv2.ServerRequest) (ServerRequestResponse, error) {
			if calls.Add(1) != 1 {
				lateCalled <- struct{}{}
				return ServerRequestResponse{}, errors.New("late handler invoked")
			}
			close(firstStarted)
			return ServerRequestResponse{}, errors.New("request handler failed")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := root.ThreadRunner().StartStream(context.Background(), StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}})
	if err != nil {
		t.Fatal(err)
	}
	<-notificationStarted
	<-firstStarted
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for stream.Next(ctx) {
	}
	if stream.Err() == nil {
		t.Fatal("request handler failure did not terminate the active stream")
	}
	if err := os.WriteFile(release, []byte("failed"), 0o600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for {
		if _, err := os.Stat(lateSent); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("fake app-server did not send the request after failure shutdown")
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case <-lateCalled:
		t.Fatal("exact handler started after failure shutdown closed admission")
	default:
	}
	close(allowNotificationFinish)
	if closeErr := root.Close(); closeErr != stream.Err() {
		t.Fatalf("Close error = %v, want first request failure %v", closeErr, stream.Err())
	}
}

func TestFailureShutdownPreservesHandlerCauseOverLaterTransportError(t *testing.T) {
	handlerCause := errors.New("handler failed first")
	release := filepath.Join(t.TempDir(), "handler-failure-observed")
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{
		CWD:     t.TempDir(),
		Command: fakeHandlerErrorThenTransportCloseCommand(release),
		ServerNotificationHandler: func(context.Context, protocolv2.ServerNotification) error {
			return handlerCause
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := root.ThreadRunner().StartStream(context.Background(), StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for stream.Next(ctx) {
	}
	if !errors.Is(stream.Err(), ErrHandlerFailed) || !errors.Is(stream.Err(), handlerCause) {
		t.Fatalf("stream error = %v, want first handler cause", stream.Err())
	}
	if err := os.WriteFile(release, []byte("release"), 0o600); err != nil {
		t.Fatal(err)
	}
	if closeErr := root.Close(); !errors.Is(closeErr, ErrHandlerFailed) || !errors.Is(closeErr, handlerCause) {
		t.Fatalf("Close error = %v, want preserved handler cause", closeErr)
	}
}

func TestProtocolFailureFinishesActiveStreamsWithIndependentPartialResults(t *testing.T) {
	release := filepath.Join(t.TempDir(), "release-protocol-failure")
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeProtocolFailureMultipleStreamsCommand(release)})
	if err != nil {
		t.Fatal(err)
	}
	first, err := root.ThreadRunner().StartStream(context.Background(), StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := root.ThreadRunner().StartStream(context.Background(), StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(release, []byte("release"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for _, stream := range []*Stream[StartedThreadRun]{first, second} {
		for stream.Next(ctx) {
		}
	}
	if first.Err() == nil || first.Err() != second.Err() {
		t.Fatalf("active stream causes = %v and %v, want one shared protocol cause", first.Err(), second.Err())
	}
	results := make([]StartedThreadRun, 0, 2)
	for index, stream := range []*Stream[StartedThreadRun]{first, second} {
		result, ok := stream.Result()
		if !ok || len(result.Run.Notifications) != 1 {
			t.Fatalf("stream %d partial result = %#v, ok=%v", index, result.Run, ok)
		}
		completed, ok := result.Run.Notifications[0].AsItemCompleted()
		wantTurnID := "turn-" + itoa(index+1)
		if !ok || completed.Params.TurnID != wantTurnID || result.Start.Thread.ID != "thread-"+itoa(index+1) {
			t.Fatalf("stream %d evidence attribution = %#v, start = %#v", index, completed.Params, result.Start.Thread)
		}
		results = append(results, result)
	}
	firstCompleted, _ := results[0].Run.Notifications[0].AsItemCompleted()
	firstMessage, _ := firstCompleted.Params.Item.AsAgentMessage()
	if firstMessage.Phase == nil || firstMessage.Phase.Value == nil {
		t.Fatalf("first partial message phase = %#v", firstMessage.Phase)
	}
	*firstMessage.Phase.Value = protocolv2.MessagePhaseFinalAnswer
	secondCompleted, _ := results[1].Run.Notifications[0].AsItemCompleted()
	secondMessage, _ := secondCompleted.Params.Item.AsAgentMessage()
	if secondMessage.Phase == nil || secondMessage.Phase.Value == nil || *secondMessage.Phase.Value != protocolv2.MessagePhaseCommentary {
		t.Fatalf("mutating first stream changed second stream evidence: %#v", secondMessage)
	}
	if closeErr := root.Close(); closeErr != first.Err() {
		t.Fatalf("Close error = %v, want shared protocol cause %v", closeErr, first.Err())
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

func TestExactRunnerStreamOrdersNotificationsAndReturnsIsolatedSnapshots(t *testing.T) {
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
	first.Start.Model = "mutated-model"
	first.Start.Thread.ID = "mutated-thread"
	if first.Start.Thread.ThreadSource == nil || first.Start.Thread.ThreadSource.Value == nil {
		t.Fatalf("start thread source = %#v", first.Start.Thread.ThreadSource)
	}
	*first.Start.Thread.ThreadSource.Value = protocolv2.ThreadSource("mutated-source")
	first.Run.Turn.ID = "mutated-turn"
	turnMessage, ok := first.Run.Turn.Items[0].AsAgentMessage()
	if !ok || turnMessage.Phase == nil || turnMessage.Phase.Value == nil {
		t.Fatalf("turn message = %#v", first.Run.Turn.Items)
	}
	*turnMessage.Phase.Value = protocolv2.MessagePhaseCommentary
	first.Run.Usage.Total.InputTokens = 999
	completed, ok := first.Run.Notifications[0].AsItemCompleted()
	if !ok {
		t.Fatalf("first notification = %#v", first.Run.Notifications[0])
	}
	notificationMessage, ok := completed.Params.Item.AsAgentMessage()
	if !ok || notificationMessage.Phase == nil || notificationMessage.Phase.Value == nil {
		t.Fatalf("notification message = %#v", completed.Params.Item)
	}
	*notificationMessage.Phase.Value = protocolv2.MessagePhaseCommentary
	second, ok := stream.Result()
	if !ok {
		t.Fatal("second result snapshot unavailable")
	}
	if second.Start.Model != "gpt-exact" || second.Start.Thread.ID == "mutated-thread" || second.Start.Thread.ThreadSource == nil || second.Start.Thread.ThreadSource.Value == nil || *second.Start.Thread.ThreadSource.Value != protocolv2.ThreadSource("user") {
		t.Fatalf("start response snapshot was mutable: %#v", second.Start)
	}
	secondTurnMessage, ok := second.Run.Turn.Items[0].AsAgentMessage()
	if second.Run.Turn.ID == "mutated-turn" || !ok || secondTurnMessage.Phase == nil || secondTurnMessage.Phase.Value == nil || *secondTurnMessage.Phase.Value != protocolv2.MessagePhaseFinalAnswer {
		t.Fatalf("turn snapshot was mutable: %#v", second.Run.Turn)
	}
	if second.Run.Usage == nil || second.Run.Usage.Total.InputTokens != 30 {
		t.Fatalf("usage snapshot was mutable: %#v", second.Run.Usage)
	}
	secondCompleted, ok := second.Run.Notifications[0].AsItemCompleted()
	secondNotificationMessage, itemOK := secondCompleted.Params.Item.AsAgentMessage()
	if len(second.Run.Notifications) != 3 || !ok || !itemOK || secondNotificationMessage.Phase == nil || secondNotificationMessage.Phase.Value == nil || *secondNotificationMessage.Phase.Value != protocolv2.MessagePhaseFinalAnswer {
		t.Fatalf("notification snapshot was mutable: %#v", second.Run.Notifications)
	}
	if len(second.Run.Diagnostics) != 0 {
		t.Fatalf("diagnostic snapshot was mutable: %#v", second.Run.Diagnostics)
	}

	second.Start.Thread.ID = "second-consumer"
	second.Run.Usage.Total.InputTokens = 777
	third, ok := stream.Result()
	if !ok || third.Start.Thread.ID == "second-consumer" || third.Run.Usage.Total.InputTokens != 30 {
		t.Fatalf("result snapshot was mutable: %#v", second.Run)
	}
}

func TestExactRunnerDiagnosticSnapshotsAreIsolated(t *testing.T) {
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("malformed-notification")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	stream, err := root.ThreadRunner().StartStream(context.Background(), StartThreadRunRequest{Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for stream.Next(ctx) {
	}
	first, ok := stream.Result()
	if !ok || len(first.Run.Diagnostics) != 1 {
		t.Fatalf("diagnostic result = %#v, ok=%v", first.Run.Diagnostics, ok)
	}
	want := first.Run.Diagnostics[0]
	first.Run.Diagnostics[0].Kind = "mutated"
	second, ok := stream.Result()
	if !ok || len(second.Run.Diagnostics) != 1 || second.Run.Diagnostics[0] != want {
		t.Fatalf("diagnostic snapshot was mutable: %#v", second.Run.Diagnostics)
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

func TestExactRunAttachPreservesPendingBeforeLiveOrder(t *testing.T) {
	c := &client{
		exactStreams:        map[string]map[*exactRunState]struct{}{},
		exactAttaching:      map[string]map[*exactRunState]struct{}{},
		pendingEvents:       map[string][]rpcNotification{},
		pendingErrors:       map[string]error{},
		pendingDiagnostics:  map[string][]DiagnosticRef{},
		pendingThreadEvents: map[string][]rpcNotification{},
	}
	state := newExactRunState(nil, "thread-1", StartedThreadRun{Start: facadeThreadStartResponse("thread-1", "model")})
	state.turnID = "turn-1"
	c.exactAttaching[state.threadID] = map[*exactRunState]struct{}{state: {}}
	c.pendingEvents[state.turnID] = []rpcNotification{{method: protocolv2.MethodThreadTokenUsageUpdated, params: map[string]any{
		"threadId": "thread-1", "turnId": "turn-1",
		"tokenUsage": map[string]any{
			"last":               map[string]any{"cachedInputTokens": 0, "inputTokens": 3, "outputTokens": 2, "reasoningOutputTokens": 1, "totalTokens": 5},
			"modelContextWindow": 100,
			"total":              map[string]any{"cachedInputTokens": 0, "inputTokens": 30, "outputTokens": 20, "reasoningOutputTokens": 10, "totalTokens": 50},
		},
	}}}
	terminal := rpcNotification{method: protocolv2.MethodTurnCompleted, params: map[string]any{
		"threadId": "thread-1",
		"turn": map[string]any{
			"id": "turn-1", "status": "completed",
			"items": []map[string]any{{"id": "answer", "type": "agentMessage", "text": "done", "phase": "final_answer"}},
		},
	}}

	published := make(chan struct{})
	releaseReplay := make(chan struct{})
	c.testAfterExactStreamPublished = func() {
		close(published)
		<-releaseReplay
	}
	attached := make(chan struct{})
	go func() {
		c.attachExactStream(state)
		close(attached)
	}()
	<-published
	liveAtGate := make(chan struct{})
	state.testAtNotificationOrderGate = func() { close(liveAtGate) }
	liveRouted := make(chan bool, 1)
	go func() {
		typed, err := exactNotification(terminal)
		if err != nil {
			t.Error(err)
			liveRouted <- false
			return
		}
		liveRouted <- c.routeExactNotification(terminal, typed)
	}()
	<-liveAtGate
	close(releaseReplay)
	<-attached
	if !<-liveRouted {
		t.Fatal("live terminal was not routed to the exact run")
	}

	stream := &Stream[StartedThreadRun]{state: state}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var kinds []protocolv2.ServerNotificationKind
	for stream.Next(ctx) {
		kinds = append(kinds, stream.Notification().Kind())
	}
	result, ok := stream.Result()
	if !ok || stream.Err() != nil {
		t.Fatalf("result ok=%v err=%v", ok, stream.Err())
	}
	want := []protocolv2.ServerNotificationKind{protocolv2.ServerNotificationKindThreadTokenUsageUpdated, protocolv2.ServerNotificationKindTurnCompleted}
	if !reflect.DeepEqual(kinds, want) {
		t.Fatalf("stream order = %#v, want %#v", kinds, want)
	}
	run := result.Run
	if len(run.Notifications) != 2 || run.Notifications[0].Kind() != want[0] || run.Notifications[1].Kind() != want[1] {
		t.Fatalf("notification evidence = %#v", run.Notifications)
	}
	if run.Usage == nil || run.Usage.Total.InputTokens != 30 || run.FinalResponse != "done" {
		t.Fatalf("derived facts = usage %#v final %q", run.Usage, run.FinalResponse)
	}
}

func TestExactTerminalDeliveryDoesNotDeadlockWithAttachment(t *testing.T) {
	c := &client{
		exactStreams:        map[string]map[*exactRunState]struct{}{},
		exactAttaching:      map[string]map[*exactRunState]struct{}{},
		pendingEvents:       map[string][]rpcNotification{},
		pendingErrors:       map[string]error{},
		pendingDiagnostics:  map[string][]DiagnosticRef{},
		pendingThreadEvents: map[string][]rpcNotification{},
	}
	state := newExactRunState(c, "thread-deadlock", StartedThreadRun{Start: facadeThreadStartResponse("thread-deadlock", "model")})
	state.turnID = "turn-deadlock"
	c.exactAttaching[state.threadID] = map[*exactRunState]struct{}{state: {}}
	terminal := rpcNotification{method: protocolv2.MethodTurnCompleted, params: map[string]any{
		"threadId": state.threadID,
		"turn": map[string]any{
			"id": state.turnID, "status": "completed",
			"items": []map[string]any{{"id": "answer", "type": "agentMessage", "text": "done", "phase": "final_answer"}},
		},
	}}
	typed, err := exactNotification(terminal)
	if err != nil {
		t.Fatal(err)
	}

	orderLocked := make(chan struct{})
	releaseDelivery := make(chan struct{})
	state.testAfterNotificationOrderLocked = func() {
		close(orderLocked)
		<-releaseDelivery
	}
	delivered := make(chan struct{})
	go func() {
		c.routeExactNotification(terminal, typed)
		close(delivered)
	}()
	<-orderLocked

	attachHasTurnLock := make(chan struct{})
	c.testBeforeExactStreamOrderGate = func() { close(attachHasTurnLock) }
	attached := make(chan struct{})
	go func() {
		c.attachExactStream(state)
		close(attached)
	}()
	<-attachHasTurnLock
	close(releaseDelivery)

	for name, done := range map[string]<-chan struct{}{"delivery": delivered, "attachment": attached} {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatalf("%s deadlocked", name)
		}
	}
	if !state.terminal || state.err != nil {
		t.Fatalf("terminal state = %v, error = %v", state.terminal, state.err)
	}
}

func TestExactTerminalBeforeAttachmentIsNotPublishedAsLive(t *testing.T) {
	c := &client{
		exactStreams:        map[string]map[*exactRunState]struct{}{},
		exactAttaching:      map[string]map[*exactRunState]struct{}{},
		pendingEvents:       map[string][]rpcNotification{},
		pendingErrors:       map[string]error{},
		pendingDiagnostics:  map[string][]DiagnosticRef{},
		pendingThreadEvents: map[string][]rpcNotification{},
	}
	state := newExactRunState(c, "thread-terminal-first", StartedThreadRun{Start: facadeThreadStartResponse("thread-terminal-first", "model")})
	state.turnID = "turn-terminal-first"
	c.exactAttaching[state.threadID] = map[*exactRunState]struct{}{state: {}}
	terminal := rpcNotification{method: protocolv2.MethodTurnCompleted, params: map[string]any{
		"threadId": state.threadID,
		"turn": map[string]any{
			"id": state.turnID, "status": "completed",
			"items": []map[string]any{{"id": "answer", "type": "agentMessage", "text": "done", "phase": "final_answer"}},
		},
	}}
	typed, err := exactNotification(terminal)
	if err != nil {
		t.Fatal(err)
	}
	if !c.routeExactNotification(terminal, typed) {
		t.Fatal("terminal notification was not attributed")
	}
	c.attachExactStream(state)
	if c.hasExactRun(state.threadID, state.turnID) {
		t.Fatal("terminal exact run was published as live")
	}
	result, ok := (&Stream[StartedThreadRun]{state: state}).Result()
	if !ok || result.Run.FinalResponse != "done" {
		t.Fatalf("terminal result = %#v, ok=%v", result.Run, ok)
	}
}

func TestExactRunTurnIDAccessIsRaceFreeAcrossAttributionAndLifecycle(t *testing.T) {
	const repetitions = 100
	for index := 0; index < repetitions; index++ {
		c := &client{
			exactStreams:        map[string]map[*exactRunState]struct{}{},
			exactAttaching:      map[string]map[*exactRunState]struct{}{},
			pendingEvents:       map[string][]rpcNotification{},
			pendingErrors:       map[string]error{},
			pendingDiagnostics:  map[string][]DiagnosticRef{},
			pendingThreadEvents: map[string][]rpcNotification{},
		}
		threadID := "thread-race-" + itoa(index)
		turnID := "turn-race-" + itoa(index)
		state := newExactRunState(c, threadID, StartedThreadRun{Start: facadeThreadStartResponse(threadID, "model")})
		c.registerAttachingExactStream(state)
		notification := rpcNotification{method: protocolv2.MethodModelRerouted, params: map[string]any{
			"threadId": threadID, "turnId": turnID, "fromModel": "a", "toModel": "b", "reason": "highRiskCyberActivity",
		}}
		typed, err := exactNotification(notification)
		if err != nil {
			t.Fatal(err)
		}

		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			<-start
			state.setTurn(protocolv2.Turn{ID: turnID, Status: protocolv2.TurnStatusInProgress})
		}()
		go func() {
			defer wg.Done()
			<-start
			c.routeExactNotification(notification, typed)
		}()
		go func() {
			defer wg.Done()
			<-start
			c.unregisterExactRun(state)
		}()
		close(start)
		wg.Wait()
	}
}

func TestExactNotificationOverflowHasTimingIndependentClientFailureSemantics(t *testing.T) {
	for _, test := range []struct {
		name string
		mode string
	}{
		{name: "live", mode: "exact-overflow-live"},
		{name: "pending replay", mode: "exact-overflow-pending"},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertExactOverflowClientContract(t, test.mode, false)
		})
	}
}

func TestExactTerminalOverflowPreservesTerminalEvidence(t *testing.T) {
	assertExactOverflowClientContract(t, "exact-overflow-terminal", true)
}

func assertExactOverflowClientContract(t *testing.T, mode string, terminal bool) {
	t.Helper()
	release := filepath.Join(t.TempDir(), "release-overflow")
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand(mode, release)})
	if err != nil {
		t.Fatal(err)
	}
	concrete := root.(*client)
	concrete.testExactStreamQueueCapacity = 1

	other, err := root.ThreadRunner().StartStream(context.Background(), StartThreadRunRequest{
		Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	overflowing, err := root.ThreadRunner().StartStream(context.Background(), StartThreadRunRequest{
		Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if mode == "exact-overflow-live" {
		if err := os.WriteFile(release, []byte("release"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	deadline := time.Now().Add(time.Second)
	for overflowing.Err() == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if overflowing.Err() == nil {
		t.Fatal("overflow did not finish the exact stream")
	}

	if !errors.Is(overflowing.Err(), ErrNotificationBackpressure) {
		t.Fatalf("overflowing stream error = %v", overflowing.Err())
	}
	if !errors.Is(other.Err(), ErrNotificationBackpressure) {
		t.Fatalf("other stream error = %v", other.Err())
	}
	if overflowing.Err() != other.Err() {
		t.Fatalf("active streams received different first causes: %v and %v", overflowing.Err(), other.Err())
	}
	result, ok := overflowing.Result()
	if !ok || len(result.Run.Notifications) != 2 {
		t.Fatalf("partial notification history = %d, ok=%v", len(result.Run.Notifications), ok)
	}
	last := result.Run.Notifications[1]
	if terminal {
		if last.Kind() != protocolv2.ServerNotificationKindTurnCompleted || result.Run.FinalResponse != "done" {
			t.Fatalf("terminal evidence = %#v, final response = %q", last, result.Run.FinalResponse)
		}
	} else {
		rerouted, ok := last.AsModelRerouted()
		if !ok || rerouted.Params.ToModel != "model-c" {
			t.Fatalf("overflowing generated fact not preserved: %#v", last)
		}
	}
	otherResult, ok := other.Result()
	if !ok || len(otherResult.Run.Notifications) != 0 {
		t.Fatalf("other partial result = %#v, ok=%v", otherResult.Run, ok)
	}
	if _, err := root.ThreadRunner().StartStream(context.Background(), StartThreadRunRequest{
		Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{}},
	}); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("new operation error = %v, want ErrClientClosed", err)
	}
	if closeErr := root.Close(); !errors.Is(closeErr, ErrNotificationBackpressure) {
		t.Fatalf("Close error = %v, want ErrNotificationBackpressure", closeErr)
	}
}

func TestExactRunAttachDoesNotSerializeUnrelatedRuns(t *testing.T) {
	c := &client{
		exactStreams:        map[string]map[*exactRunState]struct{}{},
		exactAttaching:      map[string]map[*exactRunState]struct{}{},
		pendingEvents:       map[string][]rpcNotification{},
		pendingErrors:       map[string]error{},
		pendingDiagnostics:  map[string][]DiagnosticRef{},
		pendingThreadEvents: map[string][]rpcNotification{},
	}
	attaching := newExactRunState(nil, "thread-attaching", StartedThreadRun{Start: facadeThreadStartResponse("thread-attaching", "model")})
	attaching.turnID = "turn-attaching"
	independent := newExactRunState(nil, "thread-independent", StartedThreadRun{Start: facadeThreadStartResponse("thread-independent", "model")})
	independent.turnID = "turn-independent"
	c.exactAttaching[attaching.threadID] = map[*exactRunState]struct{}{attaching: {}}
	c.exactStreams[independent.turnID] = map[*exactRunState]struct{}{independent: {}}

	published := make(chan struct{})
	releaseReplay := make(chan struct{})
	c.testAfterExactStreamPublished = func() {
		close(published)
		<-releaseReplay
	}
	attached := make(chan struct{})
	go func() {
		c.attachExactStream(attaching)
		close(attached)
	}()
	<-published
	notification := rpcNotification{method: protocolv2.MethodModelRerouted, params: map[string]any{
		"threadId": independent.threadID, "turnId": independent.turnID, "fromModel": "a", "toModel": "b", "reason": "highRiskCyberActivity",
	}}
	typed, err := exactNotification(notification)
	if err != nil {
		t.Fatal(err)
	}
	routed := make(chan bool, 1)
	go func() { routed <- c.routeExactNotification(notification, typed) }()
	select {
	case ok := <-routed:
		if !ok {
			t.Fatal("independent notification was not routed")
		}
	case <-time.After(time.Second):
		t.Fatal("unrelated exact run was serialized behind attach replay")
	}
	close(releaseReplay)
	<-attached

	stream := &Stream[StartedThreadRun]{state: independent}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if !stream.Next(ctx) || stream.Notification().Kind() != protocolv2.ServerNotificationKindModelRerouted {
		t.Fatal("independent stream did not expose its routed notification")
	}
}

func TestExactRunAttachKeepsLatestRerouteFactLive(t *testing.T) {
	c := &client{
		exactStreams:        map[string]map[*exactRunState]struct{}{},
		exactAttaching:      map[string]map[*exactRunState]struct{}{},
		pendingEvents:       map[string][]rpcNotification{},
		pendingErrors:       map[string]error{},
		pendingDiagnostics:  map[string][]DiagnosticRef{},
		pendingThreadEvents: map[string][]rpcNotification{},
	}
	state := newExactRunState(nil, "thread-1", StartedThreadRun{Start: facadeThreadStartResponse("thread-1", "model-a")})
	state.turnID = "turn-1"
	c.exactAttaching[state.threadID] = map[*exactRunState]struct{}{state: {}}
	c.pendingEvents[state.turnID] = []rpcNotification{{method: protocolv2.MethodModelRerouted, params: map[string]any{
		"threadId": state.threadID, "turnId": state.turnID, "fromModel": "model-a", "toModel": "model-b", "reason": "highRiskCyberActivity",
	}}}
	live := rpcNotification{method: protocolv2.MethodModelRerouted, params: map[string]any{
		"threadId": state.threadID, "turnId": state.turnID, "fromModel": "model-b", "toModel": "model-c", "reason": "highRiskCyberActivity",
	}}
	published := make(chan struct{})
	releaseReplay := make(chan struct{})
	c.testAfterExactStreamPublished = func() {
		close(published)
		<-releaseReplay
	}
	attached := make(chan struct{})
	go func() {
		c.attachExactStream(state)
		close(attached)
	}()
	<-published
	liveAtGate := make(chan struct{})
	state.testAtNotificationOrderGate = func() { close(liveAtGate) }
	routed := make(chan bool, 1)
	go func() {
		typed, err := exactNotification(live)
		if err != nil {
			t.Error(err)
			routed <- false
			return
		}
		routed <- c.routeExactNotification(live, typed)
	}()
	<-liveAtGate
	close(releaseReplay)
	<-attached
	if !<-routed {
		t.Fatal("live reroute was not routed")
	}

	stream := &Stream[StartedThreadRun]{state: state}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var toModels []string
	for range 2 {
		if !stream.Next(ctx) {
			t.Fatalf("stream ended after %d reroutes: %v", len(toModels), stream.Err())
		}
		rerouted, ok := stream.Notification().AsModelRerouted()
		if !ok {
			t.Fatalf("notification = %s", stream.Notification().Kind())
		}
		toModels = append(toModels, rerouted.Params.ToModel)
	}
	if !reflect.DeepEqual(toModels, []string{"model-b", "model-c"}) {
		t.Fatalf("reroute history = %#v", toModels)
	}
	result, ok := stream.Result()
	if !ok || len(result.Run.Notifications) != 2 {
		t.Fatalf("result = %#v ok=%v", result.Run, ok)
	}
	last, ok := result.Run.Notifications[1].AsModelRerouted()
	if !ok || last.Params.ToModel != "model-c" {
		t.Fatalf("effective model fact = %#v", result.Run.Notifications)
	}
}
