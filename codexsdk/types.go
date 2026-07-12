package codexsdk

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/ronhuafeng/codexsdk-go/codexsdk/protocolv2"
	"strconv"
	"strings"
	"time"
)

var (
	ErrClientClosed             = errors.New("codexsdk: client closed")
	ErrStreamClosed             = errors.New("codexsdk: stream closed")
	ErrTurnFailed               = errors.New("codexsdk: turn failed")
	ErrTurnInterrupted          = errors.New("codexsdk: turn interrupted")
	ErrNotificationBackpressure = errors.New("codexsdk: notification backpressure")
	ErrHandlerFailed            = errors.New("codexsdk: handler failed")
	ErrExactServerRequest       = errors.New("codexsdk: exact server request failed closed")
)

type ExactServerRequestError struct {
	Kind   protocolv2.ServerRequestKind
	Reason string
}

func (e *ExactServerRequestError) Error() string {
	reason := e.Reason
	if reason == "" {
		reason = "requires application data"
	}
	return "codexsdk: exact server request " + string(e.Kind) + " " + reason
}

func (e *ExactServerRequestError) Unwrap() error { return ErrExactServerRequest }

type StartThreadRunRequest struct {
	Thread protocolv2.ThreadStartParams
	Turn   protocolv2.TurnStartParams
}

type ResumeThreadRunRequest struct {
	Thread protocolv2.ThreadResumeParams
	Turn   protocolv2.TurnStartParams
}

type ThreadRunResult struct {
	Turn          protocolv2.Turn
	Usage         *protocolv2.ThreadTokenUsage
	Notifications []protocolv2.ServerNotification
	FinalResponse string
	InputStats    InputStats
	Diagnostics   []DiagnosticRef
}

type StartedThreadRun struct {
	Start protocolv2.ThreadStartResponse
	Run   ThreadRunResult
}

type ResumedThreadRun struct {
	Resume protocolv2.ThreadResumeResponse
	Run    ThreadRunResult
}

type ThreadRunner interface {
	Start(context.Context, StartThreadRunRequest) (StartedThreadRun, error)
	Resume(context.Context, ResumeThreadRunRequest) (ResumedThreadRun, error)
	StartStream(context.Context, StartThreadRunRequest) (*Stream[StartedThreadRun], error)
	ResumeStream(context.Context, ResumeThreadRunRequest) (*Stream[ResumedThreadRun], error)
}

type ServerNotificationHandler func(context.Context, protocolv2.ServerNotification) error

// ThreadClient is the v0.1 projected lifecycle API.
//
// Deprecated: use ThreadRunner.
type ThreadClient interface {
	StartThread(ctx context.Context, req StartThreadRequest) (LegacyThreadRunResult, error)
	ResumeThread(ctx context.Context, req ResumeThreadRequest) (LegacyThreadRunResult, error)
	StartThreadStream(ctx context.Context, req StartThreadRequest) (*ThreadStream, error)
	ResumeThreadStream(ctx context.Context, req ResumeThreadRequest) (*ThreadStream, error)
	ForkThread(ctx context.Context, req ForkThreadRequest) (ThreadForkResult, error)
}

type ClientOptions struct {
	CWD                       string
	Command                   []string
	Initialize                protocolv2.InitializeParams
	ServerRequestHandler      ServerRequestHandler
	ServerNotificationHandler ServerNotificationHandler
	NotificationQueueCapacity int

	// Deprecated compatibility fields. Prefer Initialize.
	ClientName                 string
	ClientTitle                string
	Capabilities               ClientCapabilities
	LegacyServerRequestHandler LegacyServerRequestHandler
}

// Deprecated: set ClientOptions.Initialize capabilities directly.
type ClientCapabilities struct {
	ExperimentalAPI bool
}

// Deprecated: use exact generated request params with ThreadRunner.
type ThreadClientOptions struct {
	DefaultModel             string
	DefaultCWD               string
	DefaultEffort            ReasoningEffort
	DefaultApprovalPolicy    ApprovalPolicy
	DefaultApprovalsReviewer ApprovalsReviewer
	DefaultEphemeral         *bool
}

// Deprecated: use protocolv2.UserInput.
type InputItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Path string `json:"path,omitempty"`
}

const (
	InputItemText = "text"
	InputItemFile = "file"
)

// Deprecated: use protocolv2.NewUserInputText.
func Text(text string) []InputItem {
	return []InputItem{{Type: InputItemText, Text: text}}
}

// Deprecated: construct exact protocolv2.UserInput values.
func TextAndFiles(text string, paths []string) []InputItem {
	items := []InputItem{{Type: InputItemText, Text: text}}
	for _, path := range paths {
		items = append(items, InputItem{Type: InputItemFile, Path: path})
	}
	return items
}

// Deprecated: use protocolv2.Value.
func Bool(value bool) *bool {
	return &value
}

// Deprecated: use protocolv2.ReasoningEffort.
type ReasoningEffort string

const (
	ReasoningEffortNone    ReasoningEffort = "none"
	ReasoningEffortMinimal ReasoningEffort = "minimal"
	ReasoningEffortLow     ReasoningEffort = "low"
	ReasoningEffortMedium  ReasoningEffort = "medium"
	ReasoningEffortHigh    ReasoningEffort = "high"
	ReasoningEffortXHigh   ReasoningEffort = "xhigh"
)

// Deprecated: use protocolv2.AskForApproval.
type ApprovalPolicy string

const (
	ApprovalPolicyUntrusted ApprovalPolicy = "untrusted"
	ApprovalPolicyOnFailure ApprovalPolicy = "on-failure"
	ApprovalPolicyOnRequest ApprovalPolicy = "on-request"
	ApprovalPolicyNever     ApprovalPolicy = "never"
)

// Deprecated: use protocolv2.ApprovalsReviewer.
type ApprovalsReviewer string

const (
	ApprovalsReviewerUser             ApprovalsReviewer = "user"
	ApprovalsReviewerAutoReview       ApprovalsReviewer = "auto_review"
	ApprovalsReviewerGuardianSubagent ApprovalsReviewer = "guardian_subagent"
)

// Deprecated: use StartThreadRunRequest.
type StartThreadRequest struct {
	Input             []InputItem
	OutputSchema      protocolv2.OutputSchema
	Ephemeral         *bool
	Model             string
	CWD               string
	Effort            ReasoningEffort
	ApprovalPolicy    ApprovalPolicy
	ApprovalsReviewer ApprovalsReviewer
}

// Deprecated: use ResumeThreadRunRequest.
type ResumeThreadRequest struct {
	ThreadID          string
	Input             []InputItem
	OutputSchema      protocolv2.OutputSchema
	Model             string
	CWD               string
	Effort            ReasoningEffort
	ApprovalPolicy    ApprovalPolicy
	ApprovalsReviewer ApprovalsReviewer
}

// Deprecated: call the generated Threads.Fork facade directly.
type ForkThreadRequest struct {
	ParentThreadID    string
	Ephemeral         *bool
	Model             string
	CWD               string
	ApprovalPolicy    ApprovalPolicy
	ApprovalsReviewer ApprovalsReviewer
}

type ServerRequestHandler func(context.Context, protocolv2.ServerRequest) (ServerRequestResponse, error)

type ServerRequestResponse struct {
	kind  protocolv2.ServerRequestKind
	value any
}

// LegacyServerRequestHandler is the v0.1 projected callback.
//
// Deprecated: use ServerRequestHandler.
type LegacyServerRequestHandler func(ctx context.Context, req ServerRequest) (LegacyServerRequestResponse, error)

type ServerRequestKind string

const (
	ServerRequestApplyPatchApproval  ServerRequestKind = "apply_patch_approval"
	ServerRequestExecCommandApproval ServerRequestKind = "exec_command_approval"
	ServerRequestCommandApproval     ServerRequestKind = "command_approval"
	ServerRequestFileChangeApproval  ServerRequestKind = "file_change_approval"
	ServerRequestPermissionsApproval ServerRequestKind = "permissions_approval"
	ServerRequestUserInput           ServerRequestKind = "user_input"
	ServerRequestMCPElicitation      ServerRequestKind = "mcp_elicitation"
	ServerRequestToolCall            ServerRequestKind = "tool_call"
	ServerRequestChatGPTAuthRefresh  ServerRequestKind = "chatgpt_auth_refresh"
	ServerRequestAttestation         ServerRequestKind = "attestation"
	ServerRequestUnsupported         ServerRequestKind = "unsupported"
)

type ServerRequest struct {
	Kind     ServerRequestKind
	Method   string
	ThreadID string
	TurnID   string
	ItemID   string
	Approval *ApprovalRequest

	ApplyPatchApproval       *protocolv2.ApplyPatchApprovalParams
	ExecCommandApproval      *protocolv2.ExecCommandApprovalParams
	CommandExecutionApproval *protocolv2.CommandExecutionRequestApprovalParams
	FileChangeApproval       *protocolv2.FileChangeRequestApprovalParams
	PermissionsApproval      *protocolv2.PermissionsRequestApprovalParams
	ChatGPTAuthTokensRefresh *protocolv2.ChatgptAuthTokensRefreshParams
	DynamicToolCall          *protocolv2.DynamicToolCallParams
	ToolRequestUserInput     *protocolv2.ToolRequestUserInputParams
	MCPElicitation           *protocolv2.McpServerElicitationRequestParams
}

// Deprecated: use opaque ServerRequestResponse constructors.
type LegacyServerRequestResponse struct {
	ApprovalDecision ApprovalDecision

	ApplyPatchApproval       *protocolv2.ApplyPatchApprovalResponse
	ExecCommandApproval      *protocolv2.ExecCommandApprovalResponse
	CommandExecutionApproval *protocolv2.CommandExecutionRequestApprovalResponse
	FileChangeApproval       *protocolv2.FileChangeRequestApprovalResponse
	PermissionsApproval      *protocolv2.PermissionsRequestApprovalResponse
	ChatGPTAuthTokensRefresh *protocolv2.ChatgptAuthTokensRefreshResponse
	DynamicToolCall          *protocolv2.DynamicToolCallResponse
	ToolRequestUserInput     *protocolv2.ToolRequestUserInputResponse
	MCPElicitation           *protocolv2.McpServerElicitationRequestResponse
}

type ApprovalRequest struct {
	Method             string
	ThreadID           string
	TurnID             string
	ItemID             string
	Reason             string
	Command            []string
	CWD                string
	AvailableDecisions []ApprovalDecision
}

type ApprovalDecision string

const (
	ApprovalAccept           ApprovalDecision = "accept"
	ApprovalAcceptForSession ApprovalDecision = "acceptForSession"
	ApprovalDecline          ApprovalDecision = "decline"
	ApprovalCancel           ApprovalDecision = "cancel"
)

// LegacyThreadRunResult is the v0.1 projected thread result.
//
// Deprecated: use ThreadRunner and ThreadRunResult.
type LegacyThreadRunResult struct {
	ThreadID                 string
	TurnID                   string
	FinalResponse            string
	Items                    []ThreadItem
	EffectiveModel           string
	EffectiveModelProvider   string
	EffectiveReasoningEffort ReasoningEffort
	Usage                    Usage
	InputStats               InputStats
	Diagnostics              []DiagnosticRef
}

type ThreadItem struct {
	ID    string
	Type  string
	Text  string
	Phase string
}

type ThreadForkResult struct {
	ThreadID                 string
	ForkedFromID             string
	Ephemeral                *bool
	EffectiveModel           string
	EffectiveModelProvider   string
	EffectiveReasoningEffort ReasoningEffort
	Diagnostics              []DiagnosticRef
}

type Usage struct {
	InputTokens           int
	CachedInputTokens     int
	OutputTokens          int
	ReasoningOutputTokens int
}

type InputStats struct {
	ItemsCount      int
	TextBytes       int
	AttachmentCount int
	InputItemsHash  string
}

type DiagnosticRef struct {
	Kind      string
	ID        string
	Path      string
	SizeBytes int64
	SHA256    string
}

type ThreadEventKind string

const (
	ThreadEventStarted           ThreadEventKind = "started"
	ThreadEventOutputDelta       ThreadEventKind = "output_delta"
	ThreadEventUsage             ThreadEventKind = "usage"
	ThreadEventDiagnostic        ThreadEventKind = "diagnostic"
	ThreadEventTurnWarning       ThreadEventKind = "turn_warning"
	ThreadEventModelRerouted     ThreadEventKind = "model_rerouted"
	ThreadEventModelVerification ThreadEventKind = "model_verification"
	ThreadEventConfigWarning     ThreadEventKind = "config_warning"
	ThreadEventCompleted         ThreadEventKind = "completed"
)

type ThreadEvent struct {
	Kind        ThreadEventKind
	ThreadID    string
	TurnID      string
	At          time.Time
	OutputDelta string
	Usage       *Usage
	Diagnostic  *DiagnosticRef
	TurnWarning *TurnWarningEvent
	Model       *ModelEvent
	Warning     *WarningEvent
	Result      *LegacyThreadRunResult
}

type TurnWarningEvent struct {
	Code      string
	Message   string
	WillRetry bool
}

type ModelEvent struct {
	FromModel     string
	ToModel       string
	Reason        string
	Verifications []string
}

type WarningEvent struct {
	Summary string
	Details string
	Path    string
}

type turnError struct {
	ThreadID string
	TurnID   string
	Status   string
	Code     string
	Message  string
}

func (e *turnError) Error() string {
	if e == nil {
		return "<nil>"
	}
	parts := []string{"codexsdk: turn failed"}
	if e.ThreadID != "" {
		parts = append(parts, "thread_id="+e.ThreadID)
	}
	if e.TurnID != "" {
		parts = append(parts, "turn_id="+e.TurnID)
	}
	if e.Status != "" {
		parts = append(parts, "status="+e.Status)
	}
	if e.Code != "" {
		parts = append(parts, "code="+e.Code)
	}
	if e.Message != "" {
		parts = append(parts, "message="+strconv.Quote(e.Message))
	}
	return strings.Join(parts, " ")
}

type turnInterruptedError struct {
	ThreadID string
	TurnID   string
	Status   string
}

func (e *turnInterruptedError) Error() string {
	if e == nil {
		return "<nil>"
	}
	parts := []string{"codexsdk: turn interrupted"}
	if e.ThreadID != "" {
		parts = append(parts, "thread_id="+e.ThreadID)
	}
	if e.TurnID != "" {
		parts = append(parts, "turn_id="+e.TurnID)
	}
	if e.Status != "" {
		parts = append(parts, "status="+e.Status)
	}
	return strings.Join(parts, " ")
}

type unsupportedServerRequestError struct {
	RequestID string
	Method    string
	Kind      ServerRequestKind
	ThreadID  string
	TurnID    string
	ItemID    string
}

func (e *unsupportedServerRequestError) Error() string {
	if e == nil {
		return "<nil>"
	}
	parts := []string{"codexsdk: unsupported server request"}
	if e.RequestID != "" {
		parts = append(parts, "request_id="+e.RequestID)
	}
	if e.Method != "" {
		parts = append(parts, "method="+e.Method)
	}
	if e.Kind != "" {
		parts = append(parts, "kind="+string(e.Kind))
	}
	if e.ThreadID != "" {
		parts = append(parts, "thread_id="+e.ThreadID)
	}
	if e.TurnID != "" {
		parts = append(parts, "turn_id="+e.TurnID)
	}
	if e.ItemID != "" {
		parts = append(parts, "item_id="+e.ItemID)
	}
	return strings.Join(parts, " ")
}

func inputStats(items []InputItem) InputStats {
	stats := InputStats{ItemsCount: len(items)}
	normalized := make([]InputItem, 0, len(items))
	for _, item := range items {
		normalized = append(normalized, item)
		stats.TextBytes += len([]byte(item.Text))
		if item.Path != "" {
			stats.AttachmentCount++
		}
	}
	raw, _ := json.Marshal(normalized)
	sum := sha256.Sum256(raw)
	stats.InputItemsHash = hex.EncodeToString(sum[:])
	return stats
}
