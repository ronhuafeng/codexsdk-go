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
	ErrClientClosed = errors.New("codexsdk: client closed")
	ErrStreamClosed = errors.New("codexsdk: stream closed")
)

type ThreadClient interface {
	StartThread(ctx context.Context, req StartThreadRequest) (ThreadRunResult, error)
	ResumeThread(ctx context.Context, req ResumeThreadRequest) (ThreadRunResult, error)
	StartThreadStream(ctx context.Context, req StartThreadRequest) (*ThreadStream, error)
	ResumeThreadStream(ctx context.Context, req ResumeThreadRequest) (*ThreadStream, error)
	ForkThread(ctx context.Context, req ForkThreadRequest) (ThreadForkResult, error)
	Close() error
}

type Client interface {
	Accounts() Accounts
	Apps() Apps
	Commands() Commands
	CollaborationModes() CollaborationModes
	Config() Config
	ConfigRequirements() ConfigRequirements
	ExperimentalFeatures() ExperimentalFeatures
	ExternalAgentConfigs() ExternalAgentConfigs
	Feedback() Feedback
	FS() FS
	FuzzyFileSearch() FuzzyFileSearch
	Hooks() Hooks
	Marketplace() Marketplace
	Memory() Memory
	MCPServers() MCPServers
	MCPServerStatus() MCPServerStatus
	Mock() Mock
	ModelProviders() ModelProviders
	Models() Models
	Plugins() Plugins
	Processes() Processes
	Reviews() Reviews
	Skills() Skills
	Threads() Threads
	Turns() Turns
	WindowsSandbox() WindowsSandbox
	ThreadClient(options ThreadClientOptions) ThreadClient
	Close() error
}

type Accounts interface {
	LoginCancel(ctx context.Context, params protocolv2.CancelLoginAccountParams) (protocolv2.CancelLoginAccountResponse, error)
	LoginStart(ctx context.Context, params protocolv2.LoginAccountParams) (protocolv2.LoginAccountResponse, error)
	Logout(ctx context.Context) (protocolv2.LogoutAccountResponse, error)
	RateLimitsRead(ctx context.Context) (protocolv2.GetAccountRateLimitsResponse, error)
	RateLimitResetCreditConsume(ctx context.Context, params protocolv2.ConsumeAccountRateLimitResetCreditParams) (protocolv2.ConsumeAccountRateLimitResetCreditResponse, error)
	Read(ctx context.Context, params protocolv2.GetAccountParams) (protocolv2.GetAccountResponse, error)
	SendAddCreditsNudgeEmail(ctx context.Context, params protocolv2.SendAddCreditsNudgeEmailParams) (protocolv2.SendAddCreditsNudgeEmailResponse, error)
}

type Apps interface {
	List(ctx context.Context, params protocolv2.AppsListParams) (protocolv2.AppsListResponse, error)
}

type Commands interface {
	Exec(ctx context.Context, params protocolv2.CommandExecParams) (protocolv2.CommandExecResponse, error)
	ExecResize(ctx context.Context, params protocolv2.CommandExecResizeParams) (protocolv2.CommandExecResizeResponse, error)
	ExecTerminate(ctx context.Context, params protocolv2.CommandExecTerminateParams) (protocolv2.CommandExecTerminateResponse, error)
	ExecWrite(ctx context.Context, params protocolv2.CommandExecWriteParams) (protocolv2.CommandExecWriteResponse, error)
}

type CollaborationModes interface {
	List(ctx context.Context, params protocolv2.CollaborationModeListParams) (protocolv2.CollaborationModeListResponse, error)
}

type Config interface {
	BatchWrite(ctx context.Context, params protocolv2.ConfigBatchWriteParams) (protocolv2.ConfigWriteResponse, error)
	MCPServerReload(ctx context.Context) (protocolv2.McpServerRefreshResponse, error)
	Read(ctx context.Context, params protocolv2.ConfigReadParams) (protocolv2.ConfigReadResponse, error)
	ValueWrite(ctx context.Context, params protocolv2.ConfigValueWriteParams) (protocolv2.ConfigWriteResponse, error)
}

type ConfigRequirements interface {
	Read(ctx context.Context) (protocolv2.ConfigRequirementsReadResponse, error)
}

type ExperimentalFeatures interface {
	EnablementSet(ctx context.Context, params protocolv2.ExperimentalFeatureEnablementSetParams) (protocolv2.ExperimentalFeatureEnablementSetResponse, error)
	List(ctx context.Context, params protocolv2.ExperimentalFeatureListParams) (protocolv2.ExperimentalFeatureListResponse, error)
}

type ExternalAgentConfigs interface {
	Detect(ctx context.Context, params protocolv2.ExternalAgentConfigDetectParams) (protocolv2.ExternalAgentConfigDetectResponse, error)
	Import(ctx context.Context, params protocolv2.ExternalAgentConfigImportParams) (protocolv2.ExternalAgentConfigImportResponse, error)
}

type Feedback interface {
	Upload(ctx context.Context, params protocolv2.FeedbackUploadParams) (protocolv2.FeedbackUploadResponse, error)
}

type FS interface {
	Copy(ctx context.Context, params protocolv2.FsCopyParams) (protocolv2.FsCopyResponse, error)
	CreateDirectory(ctx context.Context, params protocolv2.FsCreateDirectoryParams) (protocolv2.FsCreateDirectoryResponse, error)
	GetMetadata(ctx context.Context, params protocolv2.FsGetMetadataParams) (protocolv2.FsGetMetadataResponse, error)
	ReadDirectory(ctx context.Context, params protocolv2.FsReadDirectoryParams) (protocolv2.FsReadDirectoryResponse, error)
	ReadFile(ctx context.Context, params protocolv2.FsReadFileParams) (protocolv2.FsReadFileResponse, error)
	Remove(ctx context.Context, params protocolv2.FsRemoveParams) (protocolv2.FsRemoveResponse, error)
	Unwatch(ctx context.Context, params protocolv2.FsUnwatchParams) (protocolv2.FsUnwatchResponse, error)
	Watch(ctx context.Context, params protocolv2.FsWatchParams) (protocolv2.FsWatchResponse, error)
	WriteFile(ctx context.Context, params protocolv2.FsWriteFileParams) (protocolv2.FsWriteFileResponse, error)
}

type FuzzyFileSearch interface {
	Search(ctx context.Context, params protocolv2.FuzzyFileSearchParams) (protocolv2.FuzzyFileSearchResponse, error)
	SessionStart(ctx context.Context, params protocolv2.FuzzyFileSearchSessionStartParams) (protocolv2.FuzzyFileSearchSessionStartResponse, error)
	SessionStop(ctx context.Context, params protocolv2.FuzzyFileSearchSessionStopParams) (protocolv2.FuzzyFileSearchSessionStopResponse, error)
	SessionUpdate(ctx context.Context, params protocolv2.FuzzyFileSearchSessionUpdateParams) (protocolv2.FuzzyFileSearchSessionUpdateResponse, error)
}

type Hooks interface {
	List(ctx context.Context, params protocolv2.HooksListParams) (protocolv2.HooksListResponse, error)
}

type Marketplace interface {
	Add(ctx context.Context, params protocolv2.MarketplaceAddParams) (protocolv2.MarketplaceAddResponse, error)
	Remove(ctx context.Context, params protocolv2.MarketplaceRemoveParams) (protocolv2.MarketplaceRemoveResponse, error)
	Upgrade(ctx context.Context, params protocolv2.MarketplaceUpgradeParams) (protocolv2.MarketplaceUpgradeResponse, error)
}

type Memory interface {
	Reset(ctx context.Context) (protocolv2.MemoryResetResponse, error)
}

type Mock interface {
	ExperimentalMethod(ctx context.Context, params protocolv2.MockExperimentalMethodParams) (protocolv2.MockExperimentalMethodResponse, error)
}

type Plugins interface {
	Install(ctx context.Context, params protocolv2.PluginInstallParams) (protocolv2.PluginInstallResponse, error)
	List(ctx context.Context, params protocolv2.PluginListParams) (protocolv2.PluginListResponse, error)
	Read(ctx context.Context, params protocolv2.PluginReadParams) (protocolv2.PluginReadResponse, error)
	ShareDelete(ctx context.Context, params protocolv2.PluginShareDeleteParams) (protocolv2.PluginShareDeleteResponse, error)
	ShareList(ctx context.Context, params protocolv2.PluginShareListParams) (protocolv2.PluginShareListResponse, error)
	ShareSave(ctx context.Context, params protocolv2.PluginShareSaveParams) (protocolv2.PluginShareSaveResponse, error)
	ShareUpdateTargets(ctx context.Context, params protocolv2.PluginShareUpdateTargetsParams) (protocolv2.PluginShareUpdateTargetsResponse, error)
	SkillRead(ctx context.Context, params protocolv2.PluginSkillReadParams) (protocolv2.PluginSkillReadResponse, error)
	Uninstall(ctx context.Context, params protocolv2.PluginUninstallParams) (protocolv2.PluginUninstallResponse, error)
}

type Processes interface {
	Kill(ctx context.Context, params protocolv2.ProcessKillParams) (protocolv2.ProcessKillResponse, error)
	ResizePTY(ctx context.Context, params protocolv2.ProcessResizePtyParams) (protocolv2.ProcessResizePtyResponse, error)
	Spawn(ctx context.Context, params protocolv2.ProcessSpawnParams) (protocolv2.ProcessSpawnResponse, error)
	WriteStdin(ctx context.Context, params protocolv2.ProcessWriteStdinParams) (protocolv2.ProcessWriteStdinResponse, error)
}

type MCPServers interface {
	OAuthLogin(ctx context.Context, params protocolv2.McpServerOauthLoginParams) (protocolv2.McpServerOauthLoginResponse, error)
	ResourceRead(ctx context.Context, params protocolv2.McpResourceReadParams) (protocolv2.McpResourceReadResponse, error)
	ToolCall(ctx context.Context, params protocolv2.McpServerToolCallParams) (protocolv2.McpServerToolCallResponse, error)
}

type MCPServerStatus interface {
	List(ctx context.Context, params protocolv2.ListMcpServerStatusParams) (protocolv2.ListMcpServerStatusResponse, error)
}

type Models interface {
	List(ctx context.Context, params protocolv2.ModelListParams) (protocolv2.ModelListResponse, error)
}

type ModelProviders interface {
	CapabilitiesRead(ctx context.Context, params protocolv2.ModelProviderCapabilitiesReadParams) (protocolv2.ModelProviderCapabilitiesReadResponse, error)
}

type Reviews interface {
	Start(ctx context.Context, params protocolv2.ReviewStartParams) (protocolv2.ReviewStartResponse, error)
}

type Skills interface {
	ConfigWrite(ctx context.Context, params protocolv2.SkillsConfigWriteParams) (protocolv2.SkillsConfigWriteResponse, error)
	List(ctx context.Context, params protocolv2.SkillsListParams) (protocolv2.SkillsListResponse, error)
}

type Threads interface {
	ApproveGuardianDeniedAction(ctx context.Context, params protocolv2.ThreadApproveGuardianDeniedActionParams) (protocolv2.ThreadApproveGuardianDeniedActionResponse, error)
	Archive(ctx context.Context, params protocolv2.ThreadArchiveParams) (protocolv2.ThreadArchiveResponse, error)
	BackgroundTerminalsClean(ctx context.Context, params protocolv2.ThreadBackgroundTerminalsCleanParams) (protocolv2.ThreadBackgroundTerminalsCleanResponse, error)
	CompactStart(ctx context.Context, params protocolv2.ThreadCompactStartParams) (protocolv2.ThreadCompactStartResponse, error)
	DecrementElicitation(ctx context.Context, params protocolv2.ThreadDecrementElicitationParams) (protocolv2.ThreadDecrementElicitationResponse, error)
	Fork(ctx context.Context, params protocolv2.ThreadForkParams) (protocolv2.ThreadForkResponse, error)
	GoalClear(ctx context.Context, params protocolv2.ThreadGoalClearParams) (protocolv2.ThreadGoalClearResponse, error)
	GoalGet(ctx context.Context, params protocolv2.ThreadGoalGetParams) (protocolv2.ThreadGoalGetResponse, error)
	GoalSet(ctx context.Context, params protocolv2.ThreadGoalSetParams) (protocolv2.ThreadGoalSetResponse, error)
	IncrementElicitation(ctx context.Context, params protocolv2.ThreadIncrementElicitationParams) (protocolv2.ThreadIncrementElicitationResponse, error)
	InjectItems(ctx context.Context, params protocolv2.ThreadInjectItemsParams) (protocolv2.ThreadInjectItemsResponse, error)
	List(ctx context.Context, params protocolv2.ThreadListParams) (protocolv2.ThreadListResponse, error)
	LoadedList(ctx context.Context, params protocolv2.ThreadLoadedListParams) (protocolv2.ThreadLoadedListResponse, error)
	MemoryModeSet(ctx context.Context, params protocolv2.ThreadMemoryModeSetParams) (protocolv2.ThreadMemoryModeSetResponse, error)
	MetadataUpdate(ctx context.Context, params protocolv2.ThreadMetadataUpdateParams) (protocolv2.ThreadMetadataUpdateResponse, error)
	NameSet(ctx context.Context, params protocolv2.ThreadSetNameParams) (protocolv2.ThreadSetNameResponse, error)
	Read(ctx context.Context, params protocolv2.ThreadReadParams) (protocolv2.ThreadReadResponse, error)
	RealtimeAppendAudio(ctx context.Context, params protocolv2.ThreadRealtimeAppendAudioParams) (protocolv2.ThreadRealtimeAppendAudioResponse, error)
	RealtimeAppendSpeech(ctx context.Context, params protocolv2.ThreadRealtimeAppendSpeechParams) (protocolv2.ThreadRealtimeAppendSpeechResponse, error)
	RealtimeAppendText(ctx context.Context, params protocolv2.ThreadRealtimeAppendTextParams) (protocolv2.ThreadRealtimeAppendTextResponse, error)
	RealtimeListVoices(ctx context.Context, params protocolv2.ThreadRealtimeListVoicesParams) (protocolv2.ThreadRealtimeListVoicesResponse, error)
	RealtimeStart(ctx context.Context, params protocolv2.ThreadRealtimeStartParams) (protocolv2.ThreadRealtimeStartResponse, error)
	RealtimeStop(ctx context.Context, params protocolv2.ThreadRealtimeStopParams) (protocolv2.ThreadRealtimeStopResponse, error)
	Resume(ctx context.Context, params protocolv2.ThreadResumeParams) (protocolv2.ThreadResumeResponse, error)
	Rollback(ctx context.Context, params protocolv2.ThreadRollbackParams) (protocolv2.ThreadRollbackResponse, error)
	ShellCommand(ctx context.Context, params protocolv2.ThreadShellCommandParams) (protocolv2.ThreadShellCommandResponse, error)
	Start(ctx context.Context, params protocolv2.ThreadStartParams) (protocolv2.ThreadStartResponse, error)
	TurnsItemsList(ctx context.Context, params protocolv2.ThreadTurnsItemsListParams) (protocolv2.ThreadTurnsItemsListResponse, error)
	TurnsList(ctx context.Context, params protocolv2.ThreadTurnsListParams) (protocolv2.ThreadTurnsListResponse, error)
	Unarchive(ctx context.Context, params protocolv2.ThreadUnarchiveParams) (protocolv2.ThreadUnarchiveResponse, error)
	Unsubscribe(ctx context.Context, params protocolv2.ThreadUnsubscribeParams) (protocolv2.ThreadUnsubscribeResponse, error)
}

type Turns interface {
	Interrupt(ctx context.Context, params protocolv2.TurnInterruptParams) (protocolv2.TurnInterruptResponse, error)
	Start(ctx context.Context, params protocolv2.TurnStartParams) (protocolv2.TurnStartResponse, error)
	Steer(ctx context.Context, params protocolv2.TurnSteerParams) (protocolv2.TurnSteerResponse, error)
}

type WindowsSandbox interface {
	Readiness(ctx context.Context) (protocolv2.WindowsSandboxReadinessResponse, error)
	SetupStart(ctx context.Context, params protocolv2.WindowsSandboxSetupStartParams) (protocolv2.WindowsSandboxSetupStartResponse, error)
}

type ClientOptions struct {
	CWD                  string
	Command              []string
	ClientName           string
	ClientTitle          string
	Capabilities         ClientCapabilities
	ServerRequestHandler ServerRequestHandler
}

type ClientCapabilities struct {
	ExperimentalAPI bool
}

type ThreadClientOptions struct {
	DefaultModel  string
	DefaultCWD    string
	DefaultEffort ReasoningEffort
}

type InputItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Path string `json:"path,omitempty"`
}

const (
	InputItemText = "text"
	InputItemFile = "file"
)

func Text(text string) []InputItem {
	return []InputItem{{Type: InputItemText, Text: text}}
}

func TextAndFiles(text string, paths []string) []InputItem {
	items := []InputItem{{Type: InputItemText, Text: text}}
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		items = append(items, InputItem{Type: InputItemFile, Path: path})
	}
	return items
}

func Bool(value bool) *bool {
	return &value
}

type ReasoningEffort string

const (
	ReasoningEffortNone    ReasoningEffort = "none"
	ReasoningEffortMinimal ReasoningEffort = "minimal"
	ReasoningEffortLow     ReasoningEffort = "low"
	ReasoningEffortMedium  ReasoningEffort = "medium"
	ReasoningEffortHigh    ReasoningEffort = "high"
	ReasoningEffortXHigh   ReasoningEffort = "xhigh"
)

type ApprovalPolicy string

const (
	ApprovalPolicyUntrusted ApprovalPolicy = "untrusted"
	ApprovalPolicyOnFailure ApprovalPolicy = "on-failure"
	ApprovalPolicyOnRequest ApprovalPolicy = "on-request"
	ApprovalPolicyNever     ApprovalPolicy = "never"
)

type ApprovalsReviewer string

const (
	ApprovalsReviewerUser             ApprovalsReviewer = "user"
	ApprovalsReviewerAutoReview       ApprovalsReviewer = "auto_review"
	ApprovalsReviewerGuardianSubagent ApprovalsReviewer = "guardian_subagent"
)

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

type ForkThreadRequest struct {
	ParentThreadID    string
	Ephemeral         *bool
	Model             string
	CWD               string
	ApprovalPolicy    ApprovalPolicy
	ApprovalsReviewer ApprovalsReviewer
}

type ServerRequestHandler func(ctx context.Context, req ServerRequest) (ServerRequestResponse, error)

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

type ServerRequestResponse struct {
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

type ThreadRunResult struct {
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
	Result      *ThreadRunResult
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
