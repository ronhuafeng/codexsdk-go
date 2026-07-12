package codexsdk_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ronhuafeng/codexsdk-go/codexsdk"
	"github.com/ronhuafeng/codexsdk-go/codexsdk/protocolv2"
)

func ExampleThreadRunner_start() {
	ctx := context.Background()
	workspace, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	root, err := codexsdk.New(codexsdk.ClientOptions{
		CWD:     workspace,
		Command: []string{"codex", "app-server", "--listen", "stdio://"},
	})
	if err != nil {
		panic(err)
	}
	defer root.Close()

	result, err := root.ThreadRunner().Start(ctx, codexsdk.StartThreadRunRequest{
		Thread: protocolv2.ThreadStartParams{Model: protocolv2.Value("gpt-5")},
		Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{
			protocolv2.NewUserInputText(protocolv2.UserInputText{Text: "Reply briefly."}),
		}},
	})
	if err != nil {
		panic(err)
	}
	_ = result.Run.FinalResponse
	_ = result.Run.Notifications
}

func ExampleStream_Wait() {
	ctx := context.Background()
	root, err := codexsdk.New(codexsdk.ClientOptions{
		CWD:     os.TempDir(),
		Command: []string{"codex", "app-server", "--listen", "stdio://"},
	})
	if err != nil {
		panic(err)
	}
	defer root.Close()

	stream, err := root.ThreadRunner().StartStream(ctx, codexsdk.StartThreadRunRequest{
		Turn: protocolv2.TurnStartParams{Input: []protocolv2.UserInput{
			protocolv2.NewUserInputText(protocolv2.UserInputText{Text: "Reply briefly."}),
		}},
	})
	if err != nil {
		panic(err)
	}
	defer stream.Close()

	result, err := stream.Wait(ctx)
	if err != nil {
		// result still contains any exact evidence accepted before failure.
		panic(err)
	}
	_ = result.Run.Notifications
}

func ExampleThreadClient_textAndFileInputEphemeralStart() {
	ctx := context.Background()
	workspace, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	model := os.Getenv("CODEXSDK_EXAMPLE_MODEL")
	if model == "" {
		panic("CODEXSDK_EXAMPLE_MODEL is required")
	}
	root, err := codexsdk.New(codexsdk.ClientOptions{
		CWD:     workspace,
		Command: []string{"codex", "app-server", "--listen", "stdio://"},
	})
	if err != nil {
		panic(err)
	}
	defer root.Close()

	threads := root.ThreadClient(codexsdk.ThreadClientOptions{
		DefaultModel: model,
		DefaultCWD:   workspace,
	})
	result, err := threads.StartThread(ctx, codexsdk.StartThreadRequest{
		Input:     codexsdk.TextAndFiles("Summarize the attached file.", []string{filepath.Join(workspace, "README.md")}),
		Ephemeral: codexsdk.Bool(true),
		Effort:    codexsdk.ReasoningEffortMedium,
	})
	if err != nil {
		panic(err)
	}
	_ = result.FinalResponse
}

func ExampleThreadClient_streamingWithOutputSchema() {
	ctx := context.Background()
	workspace, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	model := os.Getenv("CODEXSDK_EXAMPLE_MODEL")
	if model == "" {
		panic("CODEXSDK_EXAMPLE_MODEL is required")
	}
	schema, err := protocolv2.OutputSchemaFromJSON([]byte(`{
		"type": "object",
		"required": ["answer"],
		"properties": {
			"answer": {"type": "string"}
		},
		"additionalProperties": false
	}`))
	if err != nil {
		panic(err)
	}
	root, err := codexsdk.New(codexsdk.ClientOptions{
		CWD:     workspace,
		Command: []string{"codex", "app-server", "--listen", "stdio://"},
	})
	if err != nil {
		panic(err)
	}
	defer root.Close()

	threads := root.ThreadClient(codexsdk.ThreadClientOptions{
		DefaultModel: model,
		DefaultCWD:   workspace,
	})
	stream, err := threads.StartThreadStream(ctx, codexsdk.StartThreadRequest{
		Input:        codexsdk.Text("Return a short JSON object with one answer field."),
		OutputSchema: schema,
	})
	if err != nil {
		panic(err)
	}
	defer stream.Close()

	for stream.Next(ctx) {
		event := stream.Event()
		switch event.Kind {
		case codexsdk.ThreadEventOutputDelta:
			_ = event.OutputDelta
		case codexsdk.ThreadEventCompleted:
			_ = event.Result.FinalResponse
		}
	}
	if err := stream.Err(); err != nil {
		panic(err)
	}
}

func ExampleThreadClient_resumeAndFork() {
	ctx := context.Background()
	workspace, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	model := os.Getenv("CODEXSDK_EXAMPLE_MODEL")
	if model == "" {
		panic("CODEXSDK_EXAMPLE_MODEL is required")
	}
	root, err := codexsdk.New(codexsdk.ClientOptions{
		CWD:     workspace,
		Command: []string{"codex", "app-server", "--listen", "stdio://"},
	})
	if err != nil {
		panic(err)
	}
	defer root.Close()

	threads := root.ThreadClient(codexsdk.ThreadClientOptions{
		DefaultModel: model,
		DefaultCWD:   workspace,
	})
	started, err := threads.StartThread(ctx, codexsdk.StartThreadRequest{
		Input: codexsdk.Text("Create a concise working note."),
	})
	if err != nil {
		panic(err)
	}
	resumed, err := threads.ResumeThread(ctx, codexsdk.ResumeThreadRequest{
		ThreadID: started.ThreadID,
		Input:    codexsdk.Text("Revise the note into two bullets."),
	})
	if err != nil {
		panic(err)
	}
	forked, err := threads.ForkThread(ctx, codexsdk.ForkThreadRequest{
		ParentThreadID: resumed.ThreadID,
		Ephemeral:      codexsdk.Bool(true),
	})
	if err != nil {
		panic(err)
	}
	_ = forked.ThreadID
}

func ExampleClient_approvalHandling() {
	ctx := context.Background()
	workspace, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	model := os.Getenv("CODEXSDK_EXAMPLE_MODEL")
	if model == "" {
		panic("CODEXSDK_EXAMPLE_MODEL is required")
	}
	root, err := codexsdk.New(codexsdk.ClientOptions{
		CWD:     workspace,
		Command: []string{"codex", "app-server", "--listen", "stdio://"},
		LegacyServerRequestHandler: func(ctx context.Context, req codexsdk.ServerRequest) (codexsdk.LegacyServerRequestResponse, error) {
			switch req.Kind {
			case codexsdk.ServerRequestApplyPatchApproval,
				codexsdk.ServerRequestExecCommandApproval,
				codexsdk.ServerRequestCommandApproval,
				codexsdk.ServerRequestFileChangeApproval,
				codexsdk.ServerRequestPermissionsApproval:
				return codexsdk.LegacyServerRequestResponse{ApprovalDecision: codexsdk.ApprovalDecline}, nil
			default:
				return codexsdk.LegacyServerRequestResponse{}, fmt.Errorf("unsupported server request: %s", req.Method)
			}
		},
	})
	if err != nil {
		panic(err)
	}
	defer root.Close()

	threads := root.ThreadClient(codexsdk.ThreadClientOptions{
		DefaultModel: model,
		DefaultCWD:   workspace,
	})
	result, err := threads.StartThread(ctx, codexsdk.StartThreadRequest{
		Input:          codexsdk.Text("Explain the current workspace without changing files."),
		ApprovalPolicy: codexsdk.ApprovalPolicyOnRequest,
	})
	if err != nil {
		panic(err)
	}
	_ = result.ThreadID
}
