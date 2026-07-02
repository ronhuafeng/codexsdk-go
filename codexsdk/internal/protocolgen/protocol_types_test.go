package protocolgen

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateProtocolTypesMatchesCheckedInOutput(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	generated, err := GenerateProtocolTypes(plan)
	if err != nil {
		t.Fatal(err)
	}
	generatedAgain, err := GenerateProtocolTypes(plan)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated, generatedAgain) {
		t.Fatal("generated protocol types are not reproducible")
	}
	checkedIn, err := os.ReadFile(filepath.Join("..", "..", "protocolv2", "protocol_types.gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated, checkedIn) {
		t.Fatal("generated protocol types do not match checked-in codexsdk/protocolv2/protocol_types.gen.go")
	}
}

func TestSelectFirstPassGeneratedTypes(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	selected, err := SelectFirstPassGeneratedTypes(plan)
	if err != nil {
		t.Fatal(err)
	}
	selectedByName := map[string]TypePlan{}
	for _, typ := range selected {
		selectedByName[typ.TypeName] = typ
		for _, field := range typ.Fields {
			if field.WireAllowsNull && field.Kind != FieldPlanJSONValue && !isNullableGoType(field.GoType) {
				t.Fatalf("type %s selected nullable field %s without Nullable support: %s", typ.TypeName, field.Path, field.GoType)
			}
		}
	}
	if got, min := len(selected), 439; got < min {
		t.Fatalf("selected generated type count = %d, want at least %d", got, min)
	}
	for _, name := range []string{
		"AccountLoginCompletedNotification",
		"AccountRateLimitsUpdatedNotification",
		"AccountUpdatedNotification",
		"AnalyticsConfig",
		"AppConfig",
		"ActivePermissionProfile",
		"AppBranding",
		"AppInfo",
		"AppListUpdatedNotification",
		"AppMetadata",
		"AppReview",
		"AppScreenshot",
		"AppSummary",
		"AppToolConfig",
		"AppToolsConfig",
		"ApplyPatchApprovalParams",
		"ApplyPatchApprovalResponse",
		"AppsConfig",
		"AppsDefaultConfig",
		"AppsListParams",
		"AppsListResponse",
		"ByteRange",
		"CancelLoginAccountParams",
		"CancelLoginAccountResponse",
		"ChatgptAuthTokensRefreshParams",
		"ClientInfo",
		"CollabAgentState",
		"CollaborationMode",
		"CollaborationModeListParams",
		"CollaborationModeListResponse",
		"CollaborationModeMask",
		"CommandExecOutputDeltaNotification",
		"CommandExecParams",
		"CommandExecResizeParams",
		"CommandExecResizeResponse",
		"CommandExecResponse",
		"CommandExecTerminalSize",
		"CommandExecTerminateParams",
		"CommandExecTerminateResponse",
		"CommandExecWriteParams",
		"CommandExecWriteResponse",
		"CommandExecutionOutputDeltaNotification",
		"CommandExecutionRequestApprovalParams",
		"CommandExecutionRequestApprovalResponse",
		"CommandMigration",
		"ConfigBatchWriteParams",
		"Config",
		"ConfigEdit",
		"ConfigLayer",
		"ConfigLayerMetadata",
		"ConfigReadParams",
		"ConfigReadResponse",
		"ConfigRequirements",
		"ConfigRequirementsReadResponse",
		"ConfigValueWriteParams",
		"ConfigWarningNotification",
		"ConfigWriteResponse",
		"ConfiguredHookMatcherGroup",
		"ConsumeAccountRateLimitResetCreditParams",
		"ConsumeAccountRateLimitResetCreditResponse",
		"CreditsSnapshot",
		"DynamicToolCallResponse",
		"DynamicToolCallParams",
		"ErrorNotification",
		"ExecCommandApprovalParams",
		"ExecCommandApprovalResponse",
		"ExperimentalFeatureEnablementSetParams",
		"ExperimentalFeatureEnablementSetResponse",
		"ExperimentalFeature",
		"ExperimentalFeatureListParams",
		"ExperimentalFeatureListResponse",
		"ExternalAgentConfigDetectParams",
		"ExternalAgentConfigDetectResponse",
		"ExternalAgentConfigImportItemTypeFailure",
		"ExternalAgentConfigImportItemTypeSuccess",
		"ExternalAgentConfigImportParams",
		"ExternalAgentConfigImportResponse",
		"ExternalAgentConfigImportTypeResult",
		"ExternalAgentConfigMigrationItem",
		"FeedbackUploadParams",
		"FileChangePatchUpdatedNotification",
		"FileChangeRequestApprovalParams",
		"FileChangeRequestApprovalResponse",
		"FileUpdateChange",
		"FileSystemSandboxEntry",
		"FsChangedNotification",
		"FsCopyParams",
		"FsCopyResponse",
		"FsCreateDirectoryParams",
		"FsCreateDirectoryResponse",
		"FsGetMetadataParams",
		"FsGetMetadataResponse",
		"FsReadDirectoryEntry",
		"FsReadDirectoryParams",
		"FsReadDirectoryResponse",
		"FsReadFileParams",
		"FsReadFileResponse",
		"FsRemoveParams",
		"FsRemoveResponse",
		"FsWatchParams",
		"FsWatchResponse",
		"FsWriteFileParams",
		"FsWriteFileResponse",
		"FuzzyFileSearchResponse",
		"FuzzyFileSearchResult",
		"FuzzyFileSearchSessionUpdatedNotification",
		"GetAccountParams",
		"GetAccountRateLimitsResponse",
		"GetAccountResponse",
		"GuardianApprovalReview",
		"HookErrorInfo",
		"HookMetadata",
		"InitializeCapabilities",
		"InitializeParams",
		"InitializeResponse",
		"ListMcpServerStatusParams",
		"ListMcpServerStatusResponse",
		"LogoutAccountResponse",
		"MarketplaceAddParams",
		"MarketplaceAddResponse",
		"MarketplaceInterface",
		"MarketplaceLoadErrorInfo",
		"MarketplaceRemoveParams",
		"MarketplaceRemoveResponse",
		"MarketplaceUpgradeErrorInfo",
		"MarketplaceUpgradeParams",
		"MarketplaceUpgradeResponse",
		"McpResourceReadParams",
		"McpResourceReadResponse",
		"McpServerElicitationRequestParams",
		"McpServerElicitationRequestResponse",
		"McpServerOauthLoginParams",
		"McpServerOauthLoginResponse",
		"McpServerRefreshResponse",
		"McpServerStatus",
		"McpServerToolCallParams",
		"McpServerToolCallResponse",
		"Model",
		"ModelAvailabilityNux",
		"ModelListParams",
		"ModelListResponse",
		"ModelProviderCapabilitiesReadParams",
		"ModelProviderCapabilitiesReadResponse",
		"ModelReroutedNotification",
		"ModelServiceTier",
		"ModelUpgradeInfo",
		"ModelVerificationNotification",
		"NetworkPolicyAmendment",
		"NetworkApprovalContext",
		"NetworkRequirements",
		"OverriddenMetadata",
		"GrantedPermissionProfile",
		"GitInfo",
		"HookMigration",
		"HookPromptFragment",
		"HooksListEntry",
		"HooksListParams",
		"HooksListResponse",
		"HookCompletedNotification",
		"HookOutputEntry",
		"HookRunSummary",
		"HookStartedNotification",
		"PermissionsRequestApprovalParams",
		"PermissionsRequestApprovalResponse",
		"PluginDetail",
		"PluginInstallParams",
		"PluginInstallResponse",
		"PluginHookSummary",
		"PluginInterface",
		"PluginListParams",
		"PluginListResponse",
		"PluginMarketplaceEntry",
		"PluginReadParams",
		"PluginReadResponse",
		"PluginShareContext",
		"PluginShareDeleteParams",
		"PluginShareDeleteResponse",
		"PluginShareListItem",
		"PluginShareListParams",
		"PluginShareListResponse",
		"PluginSharePrincipal",
		"PluginShareSaveParams",
		"PluginShareSaveResponse",
		"PluginShareTarget",
		"PluginShareUpdateTargetsParams",
		"PluginShareUpdateTargetsResponse",
		"PluginSkillReadParams",
		"PluginSkillReadResponse",
		"PluginSummary",
		"PluginUninstallParams",
		"PluginUninstallResponse",
		"PluginsMigration",
		"ProcessExitedNotification",
		"ProcessKillParams",
		"ProcessKillResponse",
		"ProcessOutputDeltaNotification",
		"ProcessResizePtyParams",
		"ProcessResizePtyResponse",
		"ProcessSpawnParams",
		"ProcessSpawnResponse",
		"ProcessTerminalSize",
		"ProcessWriteStdinParams",
		"ProcessWriteStdinResponse",
		"RateLimitResetCreditsSummary",
		"RateLimitSnapshot",
		"RateLimitWindow",
		"RawResponseItemCompletedNotification",
		"ReasoningEffortOption",
		"RequestPermissionProfile",
		"ReviewStartParams",
		"Resource",
		"ResourceTemplate",
		"SandboxWorkspaceWrite",
		"Settings",
		"SkillsConfigWriteParams",
		"SkillsConfigWriteResponse",
		"SkillDependencies",
		"SkillErrorInfo",
		"SkillMetadata",
		"SkillsListParams",
		"SkillsListEntry",
		"SkillsListResponse",
		"SkillInterface",
		"SkillSummary",
		"SkillToolDependency",
		"ManagedHooksRequirements",
		"McpServerMigration",
		"McpToolCallError",
		"McpToolCallResult",
		"MemoryCitation",
		"MemoryCitationEntry",
		"MigrationDetails",
		"ThreadApproveGuardianDeniedActionParams",
		"ThreadForkParams",
		"ThreadForkResponse",
		"ThreadApproveGuardianDeniedActionResponse",
		"ThreadListResponse",
		"ThreadMetadataUpdateResponse",
		"ThreadReadResponse",
		"ThreadRollbackParams",
		"ThreadRollbackResponse",
		"ThreadResumeParams",
		"ThreadResumeResponse",
		"ReviewStartResponse",
		"SendAddCreditsNudgeEmailParams",
		"SendAddCreditsNudgeEmailResponse",
		"ServerRequestResolvedNotification",
		"SessionMigration",
		"ThreadArchiveParams",
		"ThreadArchiveResponse",
		"ThreadArchivedNotification",
		"ThreadBackgroundTerminalsCleanParams",
		"ThreadBackgroundTerminalsCleanResponse",
		"ThreadClosedNotification",
		"ThreadCompactStartParams",
		"ThreadCompactStartResponse",
		"ThreadDecrementElicitationParams",
		"ThreadDecrementElicitationResponse",
		"ThreadGoalClearParams",
		"ThreadGoalClearResponse",
		"ThreadGoalClearedNotification",
		"ThreadGoal",
		"ThreadGoalGetParams",
		"ThreadGoalGetResponse",
		"ThreadGoalSetParams",
		"ThreadGoalSetResponse",
		"ThreadGoalUpdatedNotification",
		"ThreadIncrementElicitationParams",
		"ThreadIncrementElicitationResponse",
		"ThreadInjectItemsParams",
		"ThreadInjectItemsResponse",
		"ThreadListParams",
		"ThreadLoadedListParams",
		"ThreadLoadedListResponse",
		"ThreadMemoryModeSetParams",
		"ThreadMemoryModeSetResponse",
		"ThreadMetadataGitInfoUpdateParams",
		"ThreadMetadataUpdateParams",
		"ThreadNameUpdatedNotification",
		"SubagentMigration",
		"Thread",
		"ThreadReadParams",
		"ThreadRealtimeAppendAudioResponse",
		"ThreadRealtimeAppendAudioParams",
		"ThreadRealtimeAppendSpeechParams",
		"ThreadRealtimeAppendSpeechResponse",
		"ThreadRealtimeAppendTextParams",
		"ThreadRealtimeAppendTextResponse",
		"ThreadRealtimeClosedNotification",
		"ThreadRealtimeErrorNotification",
		"ThreadRealtimeItemAddedNotification",
		"ThreadRealtimeListVoicesParams",
		"ThreadRealtimeListVoicesResponse",
		"ThreadRealtimeAudioChunk",
		"RealtimeVoicesList",
		"ThreadRealtimeOutputAudioDeltaNotification",
		"ThreadRealtimeSdpNotification",
		"ThreadRealtimeStartParams",
		"ThreadRealtimeStartResponse",
		"ThreadRealtimeStartedNotification",
		"ThreadRealtimeStopParams",
		"ThreadRealtimeStopResponse",
		"ThreadRealtimeTranscriptDeltaNotification",
		"ThreadRealtimeTranscriptDoneNotification",
		"ThreadStartParams",
		"ThreadStartResponse",
		"ThreadStartedNotification",
		"ThreadStatusChangedNotification",
		"ThreadSetNameParams",
		"ThreadSetNameResponse",
		"ThreadShellCommandParams",
		"ThreadShellCommandResponse",
		"ThreadTokenUsage",
		"ThreadTokenUsageUpdatedNotification",
		"ThreadTurnsItemsListResponse",
		"ThreadTurnsItemsListParams",
		"ThreadTurnsListParams",
		"ThreadTurnsListResponse",
		"ThreadUnarchiveParams",
		"ThreadUnarchiveResponse",
		"ThreadUnarchivedNotification",
		"ThreadUnsubscribeParams",
		"ThreadUnsubscribeResponse",
		"TextPosition",
		"TokenUsageBreakdown",
		"TextRange",
		"TextElement",
		"ToolRequestUserInputAnswer",
		"Tool",
		"ToolRequestUserInputOption",
		"ToolRequestUserInputParams",
		"ToolRequestUserInputQuestion",
		"ToolRequestUserInputResponse",
		"ToolsV2",
		"Turn",
		"TurnCompletedNotification",
		"TurnEnvironmentParams",
		"TurnError",
		"TurnInterruptResponse",
		"TurnPlanStep",
		"TurnPlanUpdatedNotification",
		"TurnStartParams",
		"TurnStartResponse",
		"TurnStartedNotification",
		"TurnSteerParams",
		"W3cTraceContext",
		"WebSearchLocation",
		"WebSearchToolConfig",
		"WindowsSandboxReadinessResponse",
		"WindowsSandboxSetupStartParams",
		"AgentMessageDeltaNotification",
		"ContextCompactedNotification",
		"DeprecationNoticeNotification",
		"ExternalAgentConfigImportCompletedNotification",
		"FileChangeOutputDeltaNotification",
		"GuardianWarningNotification",
		"ItemGuardianApprovalReviewCompletedNotification",
		"ItemGuardianApprovalReviewStartedNotification",
		"McpServerOauthLoginCompletedNotification",
		"McpServerStatusUpdatedNotification",
		"McpToolCallProgressNotification",
		"PlanDeltaNotification",
		"ReasoningSummaryPartAddedNotification",
		"ReasoningSummaryTextDeltaNotification",
		"ReasoningTextDeltaNotification",
		"RemoteControlStatusChangedNotification",
		"SkillsChangedNotification",
		"TerminalInteractionNotification",
		"TurnDiffUpdatedNotification",
		"WarningNotification",
	} {
		if _, ok := selectedByName[name]; !ok {
			t.Fatalf("expected first-pass generated type %s", name)
		}
	}
	commandExec := selectedByName["CommandExecParams"]
	commandExecFields := map[string]FieldPlan{}
	for _, field := range commandExec.Fields {
		commandExecFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"command":            "[]string",
		"cwd":                "*protocolv2.Nullable[string]",
		"disableOutputCap":   "*bool",
		"disableTimeout":     "*bool",
		"env":                "*protocolv2.Nullable[map[string]*protocolv2.Nullable[string]]",
		"outputBytesCap":     "*protocolv2.Nullable[uint64]",
		"permissionProfile":  "*protocolv2.Nullable[string]",
		"processId":          "*protocolv2.Nullable[string]",
		"sandboxPolicy":      "*protocolv2.Nullable[SandboxPolicy]",
		"size":               "*protocolv2.Nullable[CommandExecTerminalSize]",
		"streamStdin":        "*bool",
		"streamStdoutStderr": "*bool",
		"timeoutMs":          "*protocolv2.Nullable[int64]",
		"tty":                "*bool",
	} {
		if got := commandExecFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("CommandExecParams.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	if commandExecFields["command"].MinItems == nil || *commandExecFields["command"].MinItems != 1 {
		t.Fatalf("CommandExecParams.command MinItems = %#v, want 1", commandExecFields["command"].MinItems)
	}
	commandExecResize := selectedByName["CommandExecResizeParams"]
	commandExecResizeFields := map[string]FieldPlan{}
	for _, field := range commandExecResize.Fields {
		commandExecResizeFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"processId": "string",
		"size":      "CommandExecTerminalSize",
	} {
		if got := commandExecResizeFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("CommandExecResizeParams.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	commandExecOutput := selectedByName["CommandExecOutputDeltaNotification"]
	commandExecOutputFields := map[string]FieldPlan{}
	for _, field := range commandExecOutput.Fields {
		commandExecOutputFields[field.FieldName] = field
	}
	if got := commandExecOutputFields["stream"].GoType; got != "CommandExecOutputStream" {
		t.Fatalf("CommandExecOutputDeltaNotification.stream GoType = %q, want CommandExecOutputStream", got)
	}
	commandExecTerminalSize := selectedByName["CommandExecTerminalSize"]
	commandExecTerminalSizeFields := map[string]FieldPlan{}
	for _, field := range commandExecTerminalSize.Fields {
		commandExecTerminalSizeFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"cols": "uint16",
		"rows": "uint16",
	} {
		if got := commandExecTerminalSizeFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("CommandExecTerminalSize.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	configBatchWrite := selectedByName["ConfigBatchWriteParams"]
	configBatchWriteFields := map[string]FieldPlan{}
	for _, field := range configBatchWrite.Fields {
		configBatchWriteFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"edits":            "[]ConfigEdit",
		"expectedVersion":  "*protocolv2.Nullable[string]",
		"filePath":         "*protocolv2.Nullable[string]",
		"reloadUserConfig": "*bool",
	} {
		if got := configBatchWriteFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("ConfigBatchWriteParams.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	configEdit := selectedByName["ConfigEdit"]
	configEditFields := map[string]FieldPlan{}
	for _, field := range configEdit.Fields {
		configEditFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"keyPath":       "string",
		"mergeStrategy": "MergeStrategy",
		"value":         "protocolv2.JSONValue",
	} {
		if got := configEditFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("ConfigEdit.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	if configEditFields["value"].Kind != FieldPlanJSONValue {
		t.Fatalf("ConfigEdit.value Kind = %s, want %s", configEditFields["value"].Kind, FieldPlanJSONValue)
	}
	configWrite := selectedByName["ConfigWriteResponse"]
	configWriteFields := map[string]FieldPlan{}
	for _, field := range configWrite.Fields {
		configWriteFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"filePath":           "string",
		"overriddenMetadata": "*protocolv2.Nullable[OverriddenMetadata]",
		"status":             "WriteStatus",
		"version":            "string",
	} {
		if got := configWriteFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("ConfigWriteResponse.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	overriddenMetadata := selectedByName["OverriddenMetadata"]
	overriddenMetadataFields := map[string]FieldPlan{}
	for _, field := range overriddenMetadata.Fields {
		overriddenMetadataFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"effectiveValue":  "protocolv2.JSONValue",
		"message":         "string",
		"overridingLayer": "ConfigLayerMetadata",
	} {
		if got := overriddenMetadataFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("OverriddenMetadata.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	if overriddenMetadataFields["effectiveValue"].Kind != FieldPlanJSONValue {
		t.Fatalf("OverriddenMetadata.effectiveValue Kind = %s, want %s", overriddenMetadataFields["effectiveValue"].Kind, FieldPlanJSONValue)
	}
	configLayerMetadata := selectedByName["ConfigLayerMetadata"]
	configLayerMetadataFields := map[string]FieldPlan{}
	for _, field := range configLayerMetadata.Fields {
		configLayerMetadataFields[field.FieldName] = field
	}
	if configLayerMetadataFields["name"].GoType != "ConfigLayerSource" || configLayerMetadataFields["version"].GoType != "string" {
		t.Fatalf("ConfigLayerMetadata fields = %#v", configLayerMetadataFields)
	}
	configReadResponse := selectedByName["ConfigReadResponse"]
	configReadResponseFields := map[string]FieldPlan{}
	for _, field := range configReadResponse.Fields {
		configReadResponseFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"config":  "Config",
		"layers":  "*protocolv2.Nullable[[]ConfigLayer]",
		"origins": "map[string]ConfigLayerMetadata",
	} {
		if got := configReadResponseFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("ConfigReadResponse.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	if configReadResponseFields["origins"].Kind != FieldPlanTypedMap {
		t.Fatalf("ConfigReadResponse.origins Kind = %s, want %s", configReadResponseFields["origins"].Kind, FieldPlanTypedMap)
	}
	appsDefaultConfig := selectedByName["AppsDefaultConfig"]
	appsDefaultConfigFields := map[string]FieldPlan{}
	for _, field := range appsDefaultConfig.Fields {
		appsDefaultConfigFields[field.FieldName] = field
	}
	if appsDefaultConfigFields["approvals_reviewer"].GoType != "*protocolv2.Nullable[ApprovalsReviewer]" ||
		appsDefaultConfigFields["approvals_reviewer"].Required {
		t.Fatalf("AppsDefaultConfig.approvals_reviewer field = %#v", appsDefaultConfigFields["approvals_reviewer"])
	}
	config := selectedByName["Config"]
	if !config.OpenDynamicProperties {
		t.Fatal("Config should preserve reviewed additionalProperties=true as dynamic JSON properties")
	}
	configFields := map[string]FieldPlan{}
	for _, field := range config.Fields {
		configFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"analytics":                            "*protocolv2.Nullable[AnalyticsConfig]",
		"approval_policy":                      "*protocolv2.Nullable[AskForApproval]",
		"approvals_reviewer":                   "*protocolv2.Nullable[ApprovalsReviewer]",
		"apps":                                 "*protocolv2.Nullable[AppsConfig]",
		"compact_prompt":                       "*protocolv2.Nullable[string]",
		"desktop":                              "*protocolv2.Nullable[map[string]protocolv2.JSONValue]",
		"developer_instructions":               "*protocolv2.Nullable[string]",
		"forced_chatgpt_workspace_id":          "*protocolv2.Nullable[ForcedChatgptWorkspaceIds]",
		"forced_login_method":                  "*protocolv2.Nullable[ForcedLoginMethod]",
		"instructions":                         "*protocolv2.Nullable[string]",
		"model":                                "*protocolv2.Nullable[string]",
		"model_auto_compact_token_limit":       "*protocolv2.Nullable[int64]",
		"model_auto_compact_token_limit_scope": "*protocolv2.Nullable[AutoCompactTokenLimitScope]",
		"model_context_window":                 "*protocolv2.Nullable[int64]",
		"model_provider":                       "*protocolv2.Nullable[string]",
		"model_reasoning_effort":               "*protocolv2.Nullable[ReasoningEffort]",
		"model_reasoning_summary":              "*protocolv2.Nullable[ReasoningSummary]",
		"model_verbosity":                      "*protocolv2.Nullable[Verbosity]",
		"review_model":                         "*protocolv2.Nullable[string]",
		"sandbox_mode":                         "*protocolv2.Nullable[SandboxMode]",
		"sandbox_workspace_write":              "*protocolv2.Nullable[SandboxWorkspaceWrite]",
		"service_tier":                         "*protocolv2.Nullable[string]",
		"tools":                                "*protocolv2.Nullable[ToolsV2]",
		"web_search":                           "*protocolv2.Nullable[WebSearchMode]",
	} {
		if got := configFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("Config.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	analyticsConfig := selectedByName["AnalyticsConfig"]
	if !analyticsConfig.OpenDynamicProperties {
		t.Fatal("AnalyticsConfig should preserve reviewed additionalProperties=true as dynamic JSON properties")
	}
	configLayer := selectedByName["ConfigLayer"]
	configLayerFields := map[string]FieldPlan{}
	for _, field := range configLayer.Fields {
		configLayerFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"config":         "protocolv2.JSONValue",
		"disabledReason": "*protocolv2.Nullable[string]",
		"name":           "ConfigLayerSource",
		"version":        "string",
	} {
		if got := configLayerFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("ConfigLayer.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	if configLayerFields["config"].Kind != FieldPlanJSONValue {
		t.Fatalf("ConfigLayer.config Kind = %s, want %s", configLayerFields["config"].Kind, FieldPlanJSONValue)
	}
	appConfig := selectedByName["AppConfig"]
	appConfigFields := map[string]FieldPlan{}
	for _, field := range appConfig.Fields {
		appConfigFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"default_tools_approval_mode": "*protocolv2.Nullable[AppToolApproval]",
		"default_tools_enabled":       "*protocolv2.Nullable[bool]",
		"destructive_enabled":         "*protocolv2.Nullable[bool]",
		"enabled":                     "*bool",
		"open_world_enabled":          "*protocolv2.Nullable[bool]",
		"tools":                       "*protocolv2.Nullable[AppToolsConfig]",
	} {
		if got := appConfigFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("AppConfig.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	sandboxWorkspaceWrite := selectedByName["SandboxWorkspaceWrite"]
	sandboxWorkspaceWriteFields := map[string]FieldPlan{}
	for _, field := range sandboxWorkspaceWrite.Fields {
		sandboxWorkspaceWriteFields[field.FieldName] = field
	}
	if sandboxWorkspaceWriteFields["writable_roots"].GoType != "*[]string" {
		t.Fatalf("SandboxWorkspaceWrite.writable_roots GoType = %q, want *[]string", sandboxWorkspaceWriteFields["writable_roots"].GoType)
	}
	configWarning := selectedByName["ConfigWarningNotification"]
	configWarningFields := map[string]FieldPlan{}
	for _, field := range configWarning.Fields {
		configWarningFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"details": "*protocolv2.Nullable[string]",
		"path":    "*protocolv2.Nullable[string]",
		"range":   "*protocolv2.Nullable[TextRange]",
		"summary": "string",
	} {
		if got := configWarningFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("ConfigWarningNotification.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	textPosition := selectedByName["TextPosition"]
	textPositionFields := map[string]FieldPlan{}
	for _, field := range textPosition.Fields {
		textPositionFields[field.FieldName] = field
	}
	if textPositionFields["column"].GoType != "uint64" || textPositionFields["line"].GoType != "uint64" {
		t.Fatalf("TextPosition fields = %#v", textPositionFields)
	}
	requirementsResponse := selectedByName["ConfigRequirementsReadResponse"]
	requirementsResponseFields := map[string]FieldPlan{}
	for _, field := range requirementsResponse.Fields {
		requirementsResponseFields[field.FieldName] = field
	}
	if requirementsResponseFields["requirements"].GoType != "*protocolv2.Nullable[ConfigRequirements]" {
		t.Fatalf("ConfigRequirementsReadResponse fields = %#v", requirementsResponseFields)
	}
	requirements := selectedByName["ConfigRequirements"]
	requirementsFields := map[string]FieldPlan{}
	for _, field := range requirements.Fields {
		requirementsFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"allowedApprovalPolicies":   "*protocolv2.Nullable[[]AskForApproval]",
		"allowedApprovalsReviewers": "*protocolv2.Nullable[[]ApprovalsReviewer]",
		"allowedSandboxModes":       "*protocolv2.Nullable[[]SandboxMode]",
		"allowedWebSearchModes":     "*protocolv2.Nullable[[]WebSearchMode]",
		"enforceResidency":          "*protocolv2.Nullable[ResidencyRequirement]",
		"featureRequirements":       "*protocolv2.Nullable[map[string]bool]",
		"hooks":                     "*protocolv2.Nullable[ManagedHooksRequirements]",
		"network":                   "*protocolv2.Nullable[NetworkRequirements]",
	} {
		if got := requirementsFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("ConfigRequirements.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	if requirementsFields["featureRequirements"].Kind != FieldPlanTypedMap {
		t.Fatalf("ConfigRequirements.featureRequirements Kind = %s, want %s", requirementsFields["featureRequirements"].Kind, FieldPlanTypedMap)
	}
	hookMatcherGroup := selectedByName["ConfiguredHookMatcherGroup"]
	hookMatcherGroupFields := map[string]FieldPlan{}
	for _, field := range hookMatcherGroup.Fields {
		hookMatcherGroupFields[field.FieldName] = field
	}
	if hookMatcherGroupFields["hooks"].GoType != "[]ConfiguredHookHandler" || hookMatcherGroupFields["matcher"].GoType != "*protocolv2.Nullable[string]" {
		t.Fatalf("ConfiguredHookMatcherGroup fields = %#v", hookMatcherGroupFields)
	}
	managedHooks := selectedByName["ManagedHooksRequirements"]
	managedHooksFields := map[string]FieldPlan{}
	for _, field := range managedHooks.Fields {
		managedHooksFields[field.FieldName] = field
	}
	for _, fieldName := range []string{"PermissionRequest", "PostCompact", "PostToolUse", "PreCompact", "PreToolUse", "SessionStart", "Stop", "UserPromptSubmit"} {
		field := managedHooksFields[fieldName]
		if field.GoType != "[]ConfiguredHookMatcherGroup" || !field.Required {
			t.Fatalf("ManagedHooksRequirements.%s field = %#v", fieldName, field)
		}
	}
	if managedHooksFields["managedDir"].GoType != "*protocolv2.Nullable[string]" || managedHooksFields["windowsManagedDir"].GoType != "*protocolv2.Nullable[string]" {
		t.Fatalf("ManagedHooksRequirements optional dir fields = %#v", managedHooksFields)
	}
	networkRequirements := selectedByName["NetworkRequirements"]
	networkRequirementsFields := map[string]FieldPlan{}
	for _, field := range networkRequirements.Fields {
		networkRequirementsFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"allowLocalBinding":                "*protocolv2.Nullable[bool]",
		"allowUnixSockets":                 "*protocolv2.Nullable[[]string]",
		"allowUpstreamProxy":               "*protocolv2.Nullable[bool]",
		"allowedDomains":                   "*protocolv2.Nullable[[]string]",
		"dangerouslyAllowAllUnixSockets":   "*protocolv2.Nullable[bool]",
		"dangerouslyAllowNonLoopbackProxy": "*protocolv2.Nullable[bool]",
		"deniedDomains":                    "*protocolv2.Nullable[[]string]",
		"domains":                          "*protocolv2.Nullable[map[string]NetworkDomainPermission]",
		"enabled":                          "*protocolv2.Nullable[bool]",
		"httpPort":                         "*protocolv2.Nullable[uint16]",
		"managedAllowedDomainsOnly":        "*protocolv2.Nullable[bool]",
		"socksPort":                        "*protocolv2.Nullable[uint16]",
		"unixSockets":                      "*protocolv2.Nullable[map[string]NetworkUnixSocketPermission]",
	} {
		if got := networkRequirementsFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("NetworkRequirements.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	if networkRequirementsFields["domains"].Kind != FieldPlanTypedMap || networkRequirementsFields["unixSockets"].Kind != FieldPlanTypedMap {
		t.Fatalf("NetworkRequirements typed map fields = %#v", networkRequirementsFields)
	}
	commandApproval := selectedByName["CommandExecutionRequestApprovalParams"]
	commandApprovalFields := map[string]FieldPlan{}
	for _, field := range commandApproval.Fields {
		commandApprovalFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"additionalPermissions":           "*protocolv2.Nullable[AdditionalPermissionProfile]",
		"availableDecisions":              "*protocolv2.Nullable[[]CommandExecutionApprovalDecision]",
		"commandActions":                  "*protocolv2.Nullable[[]CommandAction]",
		"cwd":                             "*protocolv2.Nullable[string]",
		"proposedNetworkPolicyAmendments": "*protocolv2.Nullable[[]NetworkPolicyAmendment]",
	} {
		if got := commandApprovalFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("CommandExecutionRequestApprovalParams.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	fileSystemPermissions := selectedByName["AdditionalFileSystemPermissions"]
	fileSystemPermissionFields := map[string]FieldPlan{}
	for _, field := range fileSystemPermissions.Fields {
		fileSystemPermissionFields[field.FieldName] = field
	}
	globScanMaxDepth := fileSystemPermissionFields["globScanMaxDepth"]
	if globScanMaxDepth.GoType != "*protocolv2.Nullable[uint64]" {
		t.Fatalf("AdditionalFileSystemPermissions.globScanMaxDepth GoType = %q", globScanMaxDepth.GoType)
	}
	if globScanMaxDepth.Minimum == nil || *globScanMaxDepth.Minimum != 1 {
		t.Fatalf("AdditionalFileSystemPermissions.globScanMaxDepth minimum = %#v", globScanMaxDepth.Minimum)
	}
	fuzzyResult := selectedByName["FuzzyFileSearchResult"]
	fuzzyResultFields := map[string]FieldPlan{}
	for _, field := range fuzzyResult.Fields {
		fuzzyResultFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"file_name":  "string",
		"indices":    "*protocolv2.Nullable[[]uint32]",
		"match_type": "FuzzyFileSearchMatchType",
		"path":       "string",
		"root":       "string",
		"score":      "uint32",
	} {
		if got := fuzzyResultFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("FuzzyFileSearchResult.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	if got := fuzzyResultFields["indices"].Kind; got != FieldPlanArrayScalar {
		t.Fatalf("FuzzyFileSearchResult.indices kind = %s, want %s", got, FieldPlanArrayScalar)
	}
	fsCopy := selectedByName["FsCopyParams"]
	fsCopyFields := map[string]FieldPlan{}
	for _, field := range fsCopy.Fields {
		fsCopyFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"destinationPath": "string",
		"recursive":       "*bool",
		"sourcePath":      "string",
	} {
		if got := fsCopyFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("FsCopyParams.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	if fsCopyFields["destinationPath"].Kind != FieldPlanScalar {
		t.Fatalf("FsCopyParams.destinationPath kind = %s, want %s", fsCopyFields["destinationPath"].Kind, FieldPlanScalar)
	}
	fsCreateDirectory := selectedByName["FsCreateDirectoryParams"]
	fsCreateDirectoryFields := map[string]FieldPlan{}
	for _, field := range fsCreateDirectory.Fields {
		fsCreateDirectoryFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"path":      "string",
		"recursive": "*protocolv2.Nullable[bool]",
	} {
		if got := fsCreateDirectoryFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("FsCreateDirectoryParams.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	fsReadDirectoryResponse := selectedByName["FsReadDirectoryResponse"]
	fsReadDirectoryResponseFields := map[string]FieldPlan{}
	for _, field := range fsReadDirectoryResponse.Fields {
		fsReadDirectoryResponseFields[field.FieldName] = field
	}
	if got := fsReadDirectoryResponseFields["entries"].GoType; got != "[]FsReadDirectoryEntry" {
		t.Fatalf("FsReadDirectoryResponse.entries GoType = %q, want []FsReadDirectoryEntry", got)
	}
	fsReadDirectoryEntry := selectedByName["FsReadDirectoryEntry"]
	fsReadDirectoryEntryFields := map[string]FieldPlan{}
	for _, field := range fsReadDirectoryEntry.Fields {
		fsReadDirectoryEntryFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"fileName":    "string",
		"isDirectory": "bool",
		"isFile":      "bool",
	} {
		if got := fsReadDirectoryEntryFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("FsReadDirectoryEntry.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	initializeResponse := selectedByName["InitializeResponse"]
	initializeResponseFields := map[string]FieldPlan{}
	for _, field := range initializeResponse.Fields {
		initializeResponseFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"codexHome":      "string",
		"platformFamily": "string",
		"platformOs":     "string",
		"userAgent":      "string",
	} {
		if got := initializeResponseFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("InitializeResponse.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	initializeParamsFields := map[string]FieldPlan{}
	for _, field := range selectedByName["InitializeParams"].Fields {
		initializeParamsFields[field.FieldName] = field
	}
	if initializeParamsFields["clientInfo"].GoType != "ClientInfo" || !initializeParamsFields["clientInfo"].Required {
		t.Fatalf("InitializeParams.clientInfo field = %#v", initializeParamsFields["clientInfo"])
	}
	if initializeParamsFields["capabilities"].GoType != "*protocolv2.Nullable[InitializeCapabilities]" ||
		initializeParamsFields["capabilities"].Required {
		t.Fatalf("InitializeParams.capabilities field = %#v", initializeParamsFields["capabilities"])
	}
	clientInfoFields := map[string]FieldPlan{}
	for _, field := range selectedByName["ClientInfo"].Fields {
		clientInfoFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"name":    "string",
		"title":   "*protocolv2.Nullable[string]",
		"version": "string",
	} {
		if got := clientInfoFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("ClientInfo.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	if !clientInfoFields["name"].Required || !clientInfoFields["version"].Required || clientInfoFields["title"].Required {
		t.Fatalf("ClientInfo field requiredness = %#v", clientInfoFields)
	}
	initializeCapabilitiesFields := map[string]FieldPlan{}
	for _, field := range selectedByName["InitializeCapabilities"].Fields {
		initializeCapabilitiesFields[field.FieldName] = field
	}
	if initializeCapabilitiesFields["experimentalApi"].GoType != "*bool" ||
		initializeCapabilitiesFields["optOutNotificationMethods"].GoType != "*protocolv2.Nullable[[]string]" {
		t.Fatalf("InitializeCapabilities fields = %#v", initializeCapabilitiesFields)
	}
	requestPermissions := selectedByName["PermissionsRequestApprovalParams"]
	requestPermissionFields := map[string]FieldPlan{}
	for _, field := range requestPermissions.Fields {
		requestPermissionFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"cwd":         "string",
		"permissions": "RequestPermissionProfile",
		"reason":      "*protocolv2.Nullable[string]",
		"startedAtMs": "int64",
	} {
		if got := requestPermissionFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("PermissionsRequestApprovalParams.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	responsePermissions := selectedByName["PermissionsRequestApprovalResponse"]
	responsePermissionFields := map[string]FieldPlan{}
	for _, field := range responsePermissions.Fields {
		responsePermissionFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"permissions":      "GrantedPermissionProfile",
		"scope":            "*PermissionGrantScope",
		"strictAutoReview": "*protocolv2.Nullable[bool]",
	} {
		if got := responsePermissionFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("PermissionsRequestApprovalResponse.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	for _, typeName := range []string{"RequestPermissionProfile", "GrantedPermissionProfile"} {
		profile := selectedByName[typeName]
		fields := map[string]FieldPlan{}
		for _, field := range profile.Fields {
			fields[field.FieldName] = field
		}
		for fieldName, wantGoType := range map[string]string{
			"fileSystem": "*protocolv2.Nullable[AdditionalFileSystemPermissions]",
			"network":    "*protocolv2.Nullable[AdditionalNetworkPermissions]",
		} {
			if got := fields[fieldName].GoType; got != wantGoType {
				t.Fatalf("%s.%s GoType = %q, want %q", typeName, fieldName, got, wantGoType)
			}
		}
	}
	getAccount := selectedByName["GetAccountResponse"]
	getAccountFields := map[string]FieldPlan{}
	for _, field := range getAccount.Fields {
		getAccountFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"account":            "*protocolv2.Nullable[Account]",
		"requiresOpenaiAuth": "bool",
	} {
		if got := getAccountFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("GetAccountResponse.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	rateLimitResponse := selectedByName["GetAccountRateLimitsResponse"]
	rateLimitResponseFields := map[string]FieldPlan{}
	for _, field := range rateLimitResponse.Fields {
		rateLimitResponseFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"rateLimitResetCredits": "*protocolv2.Nullable[RateLimitResetCreditsSummary]",
		"rateLimits":            "RateLimitSnapshot",
		"rateLimitsByLimitId":   "*protocolv2.Nullable[map[string]RateLimitSnapshot]",
	} {
		if got := rateLimitResponseFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("GetAccountRateLimitsResponse.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	rateLimitSnapshot := selectedByName["RateLimitSnapshot"]
	rateLimitSnapshotFields := map[string]FieldPlan{}
	for _, field := range rateLimitSnapshot.Fields {
		rateLimitSnapshotFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"credits":              "*protocolv2.Nullable[CreditsSnapshot]",
		"limitId":              "*protocolv2.Nullable[string]",
		"limitName":            "*protocolv2.Nullable[string]",
		"planType":             "*protocolv2.Nullable[PlanType]",
		"primary":              "*protocolv2.Nullable[RateLimitWindow]",
		"rateLimitReachedType": "*protocolv2.Nullable[RateLimitReachedType]",
		"secondary":            "*protocolv2.Nullable[RateLimitWindow]",
	} {
		if got := rateLimitSnapshotFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("RateLimitSnapshot.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	modelListResponse := selectedByName["ModelListResponse"]
	modelListResponseFields := map[string]FieldPlan{}
	for _, field := range modelListResponse.Fields {
		modelListResponseFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"data":       "[]Model",
		"nextCursor": "*protocolv2.Nullable[string]",
	} {
		if got := modelListResponseFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("ModelListResponse.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	model := selectedByName["Model"]
	modelFields := map[string]FieldPlan{}
	for _, field := range model.Fields {
		modelFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"defaultReasoningEffort":    "ReasoningEffort",
		"inputModalities":           "*[]InputModality",
		"serviceTiers":              "*[]ModelServiceTier",
		"supportedReasoningEfforts": "[]ReasoningEffortOption",
		"upgradeInfo":               "*protocolv2.Nullable[ModelUpgradeInfo]",
	} {
		if got := modelFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("Model.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	reasoningEffortOption := selectedByName["ReasoningEffortOption"]
	reasoningEffortOptionFields := map[string]FieldPlan{}
	for _, field := range reasoningEffortOption.Fields {
		reasoningEffortOptionFields[field.FieldName] = field
	}
	if got := reasoningEffortOptionFields["reasoningEffort"].GoType; got != "ReasoningEffort" {
		t.Fatalf("ReasoningEffortOption.reasoningEffort GoType = %q, want ReasoningEffort", got)
	}
	collaborationModeListResponse := selectedByName["CollaborationModeListResponse"]
	collaborationModeListResponseFields := map[string]FieldPlan{}
	for _, field := range collaborationModeListResponse.Fields {
		collaborationModeListResponseFields[field.FieldName] = field
	}
	if got := collaborationModeListResponseFields["data"].GoType; got != "[]CollaborationModeMask" {
		t.Fatalf("CollaborationModeListResponse.data GoType = %q, want []CollaborationModeMask", got)
	}
	collaborationModeMask := selectedByName["CollaborationModeMask"]
	collaborationModeMaskFields := map[string]FieldPlan{}
	for _, field := range collaborationModeMask.Fields {
		collaborationModeMaskFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"mode":             "*protocolv2.Nullable[ModeKind]",
		"model":            "*protocolv2.Nullable[string]",
		"name":             "string",
		"reasoning_effort": "*protocolv2.Nullable[ReasoningEffort]",
	} {
		if got := collaborationModeMaskFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("CollaborationModeMask.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	appsListResponse := selectedByName["AppsListResponse"]
	appsListResponseFields := map[string]FieldPlan{}
	for _, field := range appsListResponse.Fields {
		appsListResponseFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"data":       "[]AppInfo",
		"nextCursor": "*protocolv2.Nullable[string]",
	} {
		if got := appsListResponseFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("AppsListResponse.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	appListUpdated := selectedByName["AppListUpdatedNotification"]
	appListUpdatedFields := map[string]FieldPlan{}
	for _, field := range appListUpdated.Fields {
		appListUpdatedFields[field.FieldName] = field
	}
	if got := appListUpdatedFields["data"].GoType; got != "[]AppInfo" {
		t.Fatalf("AppListUpdatedNotification.data GoType = %q, want []AppInfo", got)
	}
	appInfo := selectedByName["AppInfo"]
	appInfoFields := map[string]FieldPlan{}
	for _, field := range appInfo.Fields {
		appInfoFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"appMetadata":        "*protocolv2.Nullable[AppMetadata]",
		"branding":           "*protocolv2.Nullable[AppBranding]",
		"id":                 "string",
		"labels":             "*protocolv2.Nullable[map[string]string]",
		"name":               "string",
		"pluginDisplayNames": "*[]string",
	} {
		if got := appInfoFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("AppInfo.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	appMetadata := selectedByName["AppMetadata"]
	appMetadataFields := map[string]FieldPlan{}
	for _, field := range appMetadata.Fields {
		appMetadataFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"categories":  "*protocolv2.Nullable[[]string]",
		"review":      "*protocolv2.Nullable[AppReview]",
		"screenshots": "*protocolv2.Nullable[[]AppScreenshot]",
	} {
		if got := appMetadataFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("AppMetadata.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	appBranding := selectedByName["AppBranding"]
	appBrandingFields := map[string]FieldPlan{}
	for _, field := range appBranding.Fields {
		appBrandingFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"isDiscoverableApp": "bool",
		"website":           "*protocolv2.Nullable[string]",
	} {
		if got := appBrandingFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("AppBranding.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	marketplaceUpgradeResponse := selectedByName["MarketplaceUpgradeResponse"]
	marketplaceUpgradeResponseFields := map[string]FieldPlan{}
	for _, field := range marketplaceUpgradeResponse.Fields {
		marketplaceUpgradeResponseFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"errors":               "[]MarketplaceUpgradeErrorInfo",
		"selectedMarketplaces": "[]string",
		"upgradedRoots":        "[]string",
	} {
		if got := marketplaceUpgradeResponseFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("MarketplaceUpgradeResponse.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	marketplaceUpgradeErrorInfo := selectedByName["MarketplaceUpgradeErrorInfo"]
	marketplaceUpgradeErrorInfoFields := map[string]FieldPlan{}
	for _, field := range marketplaceUpgradeErrorInfo.Fields {
		marketplaceUpgradeErrorInfoFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"marketplaceName": "string",
		"message":         "string",
	} {
		if got := marketplaceUpgradeErrorInfoFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("MarketplaceUpgradeErrorInfo.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	processOutput := selectedByName["ProcessOutputDeltaNotification"]
	processOutputFields := map[string]FieldPlan{}
	for _, field := range processOutput.Fields {
		processOutputFields[field.FieldName] = field
	}
	if got := processOutputFields["stream"].GoType; got != "ProcessOutputStream" {
		t.Fatalf("ProcessOutputDeltaNotification.stream GoType = %q, want ProcessOutputStream", got)
	}
	processResize := selectedByName["ProcessResizePtyParams"]
	processResizeFields := map[string]FieldPlan{}
	for _, field := range processResize.Fields {
		processResizeFields[field.FieldName] = field
	}
	if got := processResizeFields["size"].GoType; got != "ProcessTerminalSize" {
		t.Fatalf("ProcessResizePtyParams.size GoType = %q, want ProcessTerminalSize", got)
	}
	processSpawn := selectedByName["ProcessSpawnParams"]
	processSpawnFields := map[string]FieldPlan{}
	for _, field := range processSpawn.Fields {
		processSpawnFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"command":        "[]string",
		"cwd":            "string",
		"env":            "*protocolv2.Nullable[map[string]*protocolv2.Nullable[string]]",
		"outputBytesCap": "*protocolv2.Nullable[uint64]",
		"size":           "*protocolv2.Nullable[ProcessTerminalSize]",
		"timeoutMs":      "*protocolv2.Nullable[int64]",
	} {
		if got := processSpawnFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("ProcessSpawnParams.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	processTerminalSize := selectedByName["ProcessTerminalSize"]
	processTerminalSizeFields := map[string]FieldPlan{}
	for _, field := range processTerminalSize.Fields {
		processTerminalSizeFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"cols": "uint16",
		"rows": "uint16",
	} {
		if got := processTerminalSizeFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("ProcessTerminalSize.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	serverRequestResolved := selectedByName["ServerRequestResolvedNotification"]
	serverRequestResolvedFields := map[string]FieldPlan{}
	for _, field := range serverRequestResolved.Fields {
		serverRequestResolvedFields[field.FieldName] = field
	}
	if serverRequestResolvedFields["requestId"].GoType != "RequestId" ||
		!serverRequestResolvedFields["requestId"].Required ||
		serverRequestResolvedFields["threadId"].GoType != "string" ||
		!serverRequestResolvedFields["threadId"].Required {
		t.Fatalf("ServerRequestResolvedNotification fields = %#v", serverRequestResolvedFields)
	}
	threadStart := selectedByName["ThreadStartParams"]
	threadStartFields := map[string]FieldPlan{}
	for _, field := range threadStart.Fields {
		threadStartFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"config":             "*protocolv2.Nullable[map[string]protocolv2.JSONValue]",
		"dynamicTools":       "*protocolv2.Nullable[[]DynamicToolSpec]",
		"environments":       "*protocolv2.Nullable[[]TurnEnvironmentParams]",
		"permissions":        "*protocolv2.Nullable[string]",
		"serviceTier":        "*protocolv2.Nullable[string]",
		"sessionStartSource": "*protocolv2.Nullable[ThreadStartSource]",
		"threadSource":       "*protocolv2.Nullable[ThreadSource]",
	} {
		if got := threadStartFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("ThreadStartParams.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	if threadStartFields["config"].Kind != FieldPlanJSONValueMap {
		t.Fatalf("ThreadStartParams.config Kind = %s, want %s", threadStartFields["config"].Kind, FieldPlanJSONValueMap)
	}
	turnStart := selectedByName["TurnStartParams"]
	turnStartFields := map[string]FieldPlan{}
	for _, field := range turnStart.Fields {
		turnStartFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"collaborationMode":          "*protocolv2.Nullable[CollaborationMode]",
		"environments":               "*protocolv2.Nullable[[]TurnEnvironmentParams]",
		"input":                      "[]UserInput",
		"outputSchema":               "*protocolv2.OutputSchema",
		"permissions":                "*protocolv2.Nullable[string]",
		"responsesapiClientMetadata": "*protocolv2.Nullable[map[string]string]",
		"sandboxPolicy":              "*protocolv2.Nullable[SandboxPolicy]",
		"summary":                    "*protocolv2.Nullable[ReasoningSummary]",
		"threadId":                   "string",
	} {
		if got := turnStartFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("TurnStartParams.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	if turnStartFields["outputSchema"].Kind != FieldPlanOutputSchema {
		t.Fatalf("TurnStartParams.outputSchema Kind = %s, want %s", turnStartFields["outputSchema"].Kind, FieldPlanOutputSchema)
	}
	if !turnStartFields["input"].Required || !turnStartFields["threadId"].Required {
		t.Fatal("TurnStartParams input and threadId must remain required")
	}
	threadFork := selectedByName["ThreadForkParams"]
	threadForkFields := map[string]FieldPlan{}
	for _, field := range threadFork.Fields {
		threadForkFields[field.FieldName] = field
	}
	if threadForkFields["threadId"].GoType != "string" || !threadForkFields["threadId"].Required {
		t.Fatalf("ThreadForkParams.threadId = %#v", threadForkFields["threadId"])
	}
	if threadForkFields["config"].GoType != "*protocolv2.Nullable[map[string]protocolv2.JSONValue]" {
		t.Fatalf("ThreadForkParams.config GoType = %q, want Nullable JSONValue map", threadForkFields["config"].GoType)
	}
	threadResume := selectedByName["ThreadResumeParams"]
	threadResumeFields := map[string]FieldPlan{}
	for _, field := range threadResume.Fields {
		threadResumeFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"config":      "*protocolv2.Nullable[map[string]protocolv2.JSONValue]",
		"history":     "*protocolv2.Nullable[[]ResponseItem]",
		"permissions": "*protocolv2.Nullable[string]",
		"serviceTier": "*protocolv2.Nullable[string]",
		"threadId":    "string",
	} {
		if got := threadResumeFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("ThreadResumeParams.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	if !threadResumeFields["threadId"].Required {
		t.Fatal("ThreadResumeParams threadId must remain required")
	}
	if threadResumeFields["config"].Kind != FieldPlanJSONValueMap {
		t.Fatalf("ThreadResumeParams.config Kind = %s, want %s", threadResumeFields["config"].Kind, FieldPlanJSONValueMap)
	}
	threadApproveGuardianDeniedActionFields := map[string]FieldPlan{}
	for _, field := range selectedByName["ThreadApproveGuardianDeniedActionParams"].Fields {
		threadApproveGuardianDeniedActionFields[field.FieldName] = field
	}
	if threadApproveGuardianDeniedActionFields["event"].GoType != "protocolv2.JSONValue" ||
		threadApproveGuardianDeniedActionFields["event"].Kind != FieldPlanJSONValue ||
		!threadApproveGuardianDeniedActionFields["event"].Required ||
		threadApproveGuardianDeniedActionFields["threadId"].GoType != "string" ||
		!threadApproveGuardianDeniedActionFields["threadId"].Required {
		t.Fatalf("ThreadApproveGuardianDeniedActionParams fields = %#v", threadApproveGuardianDeniedActionFields)
	}
	generatedFields := func(typeName string) map[string]FieldPlan {
		fields := map[string]FieldPlan{}
		for _, field := range selectedByName[typeName].Fields {
			fields[field.FieldName] = field
		}
		return fields
	}
	w3cTraceFields := generatedFields("W3cTraceContext")
	if w3cTraceFields["traceparent"].GoType != "*protocolv2.Nullable[string]" ||
		w3cTraceFields["traceparent"].Required ||
		w3cTraceFields["tracestate"].GoType != "*protocolv2.Nullable[string]" ||
		w3cTraceFields["tracestate"].Required {
		t.Fatalf("W3cTraceContext fields = %#v", w3cTraceFields)
	}
	for _, typeName := range []string{
		"ThreadRealtimeAppendAudioResponse",
		"ThreadRealtimeAppendSpeechResponse",
		"ThreadRealtimeAppendTextResponse",
		"ThreadRealtimeListVoicesParams",
		"ThreadRealtimeStartResponse",
		"ThreadRealtimeStopResponse",
	} {
		if fields := selectedByName[typeName].Fields; len(fields) != 0 {
			t.Fatalf("%s fields = %#v, want empty realtime payload", typeName, fields)
		}
	}
	for _, typeName := range []string{
		"ThreadRealtimeAppendSpeechParams",
		"ThreadRealtimeAppendTextParams",
		"ThreadRealtimeErrorNotification",
		"ThreadRealtimeSdpNotification",
		"ThreadRealtimeStopParams",
		"ThreadRealtimeTranscriptDeltaNotification",
		"ThreadRealtimeTranscriptDoneNotification",
	} {
		fields := generatedFields(typeName)
		if fields["threadId"].GoType != "string" || !fields["threadId"].Required {
			t.Fatalf("%s.threadId field = %#v", typeName, fields["threadId"])
		}
	}
	threadRealtimeAppendTextFields := generatedFields("ThreadRealtimeAppendTextParams")
	if threadRealtimeAppendTextFields["text"].GoType != "string" || !threadRealtimeAppendTextFields["text"].Required {
		t.Fatalf("ThreadRealtimeAppendTextParams.text field = %#v", threadRealtimeAppendTextFields["text"])
	}
	threadRealtimeAppendSpeechFields := generatedFields("ThreadRealtimeAppendSpeechParams")
	if threadRealtimeAppendSpeechFields["text"].GoType != "string" || !threadRealtimeAppendSpeechFields["text"].Required {
		t.Fatalf("ThreadRealtimeAppendSpeechParams.text field = %#v", threadRealtimeAppendSpeechFields["text"])
	}
	realtimeVoicesListFields := generatedFields("RealtimeVoicesList")
	if realtimeVoicesListFields["defaultV1"].GoType != "RealtimeVoice" ||
		!realtimeVoicesListFields["defaultV1"].Required ||
		realtimeVoicesListFields["defaultV2"].GoType != "RealtimeVoice" ||
		!realtimeVoicesListFields["defaultV2"].Required ||
		realtimeVoicesListFields["v1"].GoType != "[]RealtimeVoice" ||
		!realtimeVoicesListFields["v1"].Required ||
		realtimeVoicesListFields["v2"].GoType != "[]RealtimeVoice" ||
		!realtimeVoicesListFields["v2"].Required {
		t.Fatalf("RealtimeVoicesList fields = %#v", realtimeVoicesListFields)
	}
	threadRealtimeClosedFields := generatedFields("ThreadRealtimeClosedNotification")
	if threadRealtimeClosedFields["reason"].GoType != "*protocolv2.Nullable[string]" ||
		threadRealtimeClosedFields["reason"].Required ||
		threadRealtimeClosedFields["threadId"].GoType != "string" ||
		!threadRealtimeClosedFields["threadId"].Required {
		t.Fatalf("ThreadRealtimeClosedNotification fields = %#v", threadRealtimeClosedFields)
	}
	threadRealtimeErrorFields := generatedFields("ThreadRealtimeErrorNotification")
	if threadRealtimeErrorFields["message"].GoType != "string" || !threadRealtimeErrorFields["message"].Required {
		t.Fatalf("ThreadRealtimeErrorNotification.message field = %#v", threadRealtimeErrorFields["message"])
	}
	threadRealtimeItemAddedFields := generatedFields("ThreadRealtimeItemAddedNotification")
	if threadRealtimeItemAddedFields["item"].GoType != "protocolv2.JSONValue" ||
		threadRealtimeItemAddedFields["item"].Kind != FieldPlanJSONValue ||
		!threadRealtimeItemAddedFields["item"].Required ||
		threadRealtimeItemAddedFields["threadId"].GoType != "string" ||
		!threadRealtimeItemAddedFields["threadId"].Required {
		t.Fatalf("ThreadRealtimeItemAddedNotification fields = %#v", threadRealtimeItemAddedFields)
	}
	threadRealtimeListVoicesResponseFields := generatedFields("ThreadRealtimeListVoicesResponse")
	if threadRealtimeListVoicesResponseFields["voices"].GoType != "RealtimeVoicesList" ||
		!threadRealtimeListVoicesResponseFields["voices"].Required {
		t.Fatalf("ThreadRealtimeListVoicesResponse fields = %#v", threadRealtimeListVoicesResponseFields)
	}
	threadRealtimeAudioChunkFields := generatedFields("ThreadRealtimeAudioChunk")
	if threadRealtimeAudioChunkFields["data"].GoType != "string" ||
		!threadRealtimeAudioChunkFields["data"].Required ||
		threadRealtimeAudioChunkFields["itemId"].GoType != "*protocolv2.Nullable[string]" ||
		threadRealtimeAudioChunkFields["numChannels"].GoType != "uint16" ||
		!threadRealtimeAudioChunkFields["numChannels"].Required ||
		threadRealtimeAudioChunkFields["sampleRate"].GoType != "uint32" ||
		!threadRealtimeAudioChunkFields["sampleRate"].Required ||
		threadRealtimeAudioChunkFields["samplesPerChannel"].GoType != "*protocolv2.Nullable[uint32]" {
		t.Fatalf("ThreadRealtimeAudioChunk fields = %#v", threadRealtimeAudioChunkFields)
	}
	threadRealtimeAppendAudioFields := generatedFields("ThreadRealtimeAppendAudioParams")
	if threadRealtimeAppendAudioFields["audio"].GoType != "ThreadRealtimeAudioChunk" ||
		!threadRealtimeAppendAudioFields["audio"].Required ||
		threadRealtimeAppendAudioFields["threadId"].GoType != "string" ||
		!threadRealtimeAppendAudioFields["threadId"].Required {
		t.Fatalf("ThreadRealtimeAppendAudioParams fields = %#v", threadRealtimeAppendAudioFields)
	}
	threadRealtimeOutputAudioFields := generatedFields("ThreadRealtimeOutputAudioDeltaNotification")
	if threadRealtimeOutputAudioFields["audio"].GoType != "ThreadRealtimeAudioChunk" ||
		!threadRealtimeOutputAudioFields["audio"].Required ||
		threadRealtimeOutputAudioFields["threadId"].GoType != "string" ||
		!threadRealtimeOutputAudioFields["threadId"].Required {
		t.Fatalf("ThreadRealtimeOutputAudioDeltaNotification fields = %#v", threadRealtimeOutputAudioFields)
	}
	threadRealtimeSdpFields := generatedFields("ThreadRealtimeSdpNotification")
	if threadRealtimeSdpFields["sdp"].GoType != "string" || !threadRealtimeSdpFields["sdp"].Required {
		t.Fatalf("ThreadRealtimeSdpNotification.sdp field = %#v", threadRealtimeSdpFields["sdp"])
	}
	threadRealtimeStartFields := generatedFields("ThreadRealtimeStartParams")
	if architecture, ok := threadRealtimeStartFields["architecture"]; ok &&
		(architecture.GoType != "*protocolv2.Nullable[RealtimeConversationArchitecture]" || architecture.Required) {
		t.Fatalf("ThreadRealtimeStartParams.architecture field = %#v", architecture)
	}
	if threadRealtimeStartFields["codexResponseItemPrefix"].GoType != "*protocolv2.Nullable[string]" ||
		threadRealtimeStartFields["codexResponseItemPrefix"].Required ||
		threadRealtimeStartFields["codexResponsesAsItems"].GoType != "*protocolv2.Nullable[bool]" ||
		threadRealtimeStartFields["codexResponsesAsItems"].Required ||
		threadRealtimeStartFields["includeStartupContext"].GoType != "*protocolv2.Nullable[bool]" ||
		threadRealtimeStartFields["includeStartupContext"].Required ||
		threadRealtimeStartFields["outputModality"].GoType != "RealtimeOutputModality" ||
		!threadRealtimeStartFields["outputModality"].Required ||
		threadRealtimeStartFields["prompt"].GoType != "*protocolv2.Nullable[string]" ||
		threadRealtimeStartFields["prompt"].Required ||
		threadRealtimeStartFields["realtimeSessionId"].GoType != "*protocolv2.Nullable[string]" ||
		threadRealtimeStartFields["realtimeSessionId"].Required ||
		threadRealtimeStartFields["threadId"].GoType != "string" ||
		!threadRealtimeStartFields["threadId"].Required ||
		threadRealtimeStartFields["transport"].GoType != "*protocolv2.Nullable[ThreadRealtimeStartTransport]" ||
		threadRealtimeStartFields["transport"].Required ||
		threadRealtimeStartFields["version"].GoType != "*protocolv2.Nullable[RealtimeConversationVersion]" ||
		threadRealtimeStartFields["version"].Required ||
		threadRealtimeStartFields["voice"].GoType != "*protocolv2.Nullable[RealtimeVoice]" ||
		threadRealtimeStartFields["voice"].Required {
		t.Fatalf("ThreadRealtimeStartParams fields = %#v", threadRealtimeStartFields)
	}
	threadRealtimeStartedFields := generatedFields("ThreadRealtimeStartedNotification")
	if threadRealtimeStartedFields["realtimeSessionId"].GoType != "*protocolv2.Nullable[string]" ||
		threadRealtimeStartedFields["realtimeSessionId"].Required ||
		threadRealtimeStartedFields["threadId"].GoType != "string" ||
		!threadRealtimeStartedFields["threadId"].Required ||
		threadRealtimeStartedFields["version"].GoType != "RealtimeConversationVersion" ||
		!threadRealtimeStartedFields["version"].Required {
		t.Fatalf("ThreadRealtimeStartedNotification fields = %#v", threadRealtimeStartedFields)
	}
	threadRealtimeTranscriptDeltaFields := generatedFields("ThreadRealtimeTranscriptDeltaNotification")
	for _, fieldName := range []string{"delta", "role"} {
		if threadRealtimeTranscriptDeltaFields[fieldName].GoType != "string" || !threadRealtimeTranscriptDeltaFields[fieldName].Required {
			t.Fatalf("ThreadRealtimeTranscriptDeltaNotification.%s field = %#v", fieldName, threadRealtimeTranscriptDeltaFields[fieldName])
		}
	}
	threadRealtimeTranscriptDoneFields := generatedFields("ThreadRealtimeTranscriptDoneNotification")
	for _, fieldName := range []string{"role", "text"} {
		if threadRealtimeTranscriptDoneFields[fieldName].GoType != "string" || !threadRealtimeTranscriptDoneFields[fieldName].Required {
			t.Fatalf("ThreadRealtimeTranscriptDoneNotification.%s field = %#v", fieldName, threadRealtimeTranscriptDoneFields[fieldName])
		}
	}
	for _, typeName := range []string{
		"ThreadArchiveParams",
		"ThreadArchivedNotification",
		"ThreadBackgroundTerminalsCleanParams",
		"ThreadClosedNotification",
		"ThreadCompactStartParams",
		"ThreadDecrementElicitationParams",
		"ThreadGoalClearParams",
		"ThreadGoalClearedNotification",
		"ThreadGoalGetParams",
		"ThreadIncrementElicitationParams",
		"ThreadUnarchiveParams",
		"ThreadUnarchivedNotification",
		"ThreadUnsubscribeParams",
	} {
		fields := generatedFields(typeName)
		if fields["threadId"].GoType != "string" || !fields["threadId"].Required {
			t.Fatalf("%s.threadId field = %#v", typeName, fields["threadId"])
		}
	}
	for _, typeName := range []string{
		"ThreadArchiveResponse",
		"ThreadBackgroundTerminalsCleanResponse",
		"ThreadCompactStartResponse",
		"ThreadInjectItemsResponse",
		"ThreadMemoryModeSetResponse",
		"ThreadSetNameResponse",
		"ThreadShellCommandResponse",
	} {
		if fields := selectedByName[typeName].Fields; len(fields) != 0 {
			t.Fatalf("%s fields = %#v, want empty response payload", typeName, fields)
		}
	}
	threadInjectItemsFields := generatedFields("ThreadInjectItemsParams")
	if threadInjectItemsFields["items"].GoType != "[]protocolv2.JSONValue" ||
		threadInjectItemsFields["items"].Kind != FieldPlanArrayJSONValue ||
		!threadInjectItemsFields["items"].Required {
		t.Fatalf("ThreadInjectItemsParams.items field = %#v", threadInjectItemsFields["items"])
	}
	if threadInjectItemsFields["threadId"].GoType != "string" || !threadInjectItemsFields["threadId"].Required {
		t.Fatalf("ThreadInjectItemsParams.threadId field = %#v", threadInjectItemsFields["threadId"])
	}
	threadLoadedListParamsFields := generatedFields("ThreadLoadedListParams")
	if threadLoadedListParamsFields["cursor"].GoType != "*protocolv2.Nullable[string]" ||
		threadLoadedListParamsFields["limit"].GoType != "*protocolv2.Nullable[uint32]" {
		t.Fatalf("ThreadLoadedListParams fields = %#v", threadLoadedListParamsFields)
	}
	threadListParamsFields := generatedFields("ThreadListParams")
	for fieldName, wantGoType := range map[string]string{
		"archived":       "*protocolv2.Nullable[bool]",
		"cursor":         "*protocolv2.Nullable[string]",
		"cwd":            "*protocolv2.Nullable[ThreadListCwdFilter]",
		"limit":          "*protocolv2.Nullable[uint32]",
		"modelProviders": "*protocolv2.Nullable[[]string]",
		"searchTerm":     "*protocolv2.Nullable[string]",
		"sortDirection":  "*protocolv2.Nullable[SortDirection]",
		"sortKey":        "*protocolv2.Nullable[ThreadSortKey]",
		"sourceKinds":    "*protocolv2.Nullable[[]ThreadSourceKind]",
		"useStateDbOnly": "*bool",
	} {
		if got := threadListParamsFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("ThreadListParams.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	threadLoadedListResponseFields := generatedFields("ThreadLoadedListResponse")
	if threadLoadedListResponseFields["data"].GoType != "[]string" || !threadLoadedListResponseFields["data"].Required ||
		threadLoadedListResponseFields["nextCursor"].GoType != "*protocolv2.Nullable[string]" {
		t.Fatalf("ThreadLoadedListResponse fields = %#v", threadLoadedListResponseFields)
	}
	threadMetadataGitInfoUpdateFields := generatedFields("ThreadMetadataGitInfoUpdateParams")
	for _, fieldName := range []string{"branch", "originUrl", "sha"} {
		if threadMetadataGitInfoUpdateFields[fieldName].GoType != "*protocolv2.Nullable[string]" ||
			threadMetadataGitInfoUpdateFields[fieldName].Required {
			t.Fatalf("ThreadMetadataGitInfoUpdateParams.%s field = %#v", fieldName, threadMetadataGitInfoUpdateFields[fieldName])
		}
	}
	threadMetadataUpdateFields := generatedFields("ThreadMetadataUpdateParams")
	if threadMetadataUpdateFields["gitInfo"].GoType != "*protocolv2.Nullable[ThreadMetadataGitInfoUpdateParams]" ||
		threadMetadataUpdateFields["gitInfo"].Required ||
		threadMetadataUpdateFields["threadId"].GoType != "string" ||
		!threadMetadataUpdateFields["threadId"].Required {
		t.Fatalf("ThreadMetadataUpdateParams fields = %#v", threadMetadataUpdateFields)
	}
	for _, typeName := range []string{"ThreadDecrementElicitationResponse", "ThreadIncrementElicitationResponse"} {
		fields := generatedFields(typeName)
		if fields["count"].GoType != "uint64" || !fields["count"].Required ||
			fields["paused"].GoType != "bool" || !fields["paused"].Required {
			t.Fatalf("%s fields = %#v", typeName, fields)
		}
	}
	threadGoalClearResponseFields := generatedFields("ThreadGoalClearResponse")
	if threadGoalClearResponseFields["cleared"].GoType != "bool" || !threadGoalClearResponseFields["cleared"].Required {
		t.Fatalf("ThreadGoalClearResponse.cleared field = %#v", threadGoalClearResponseFields["cleared"])
	}
	threadGoalFields := generatedFields("ThreadGoal")
	for _, fieldName := range []string{"createdAt", "timeUsedSeconds", "tokensUsed", "updatedAt"} {
		if threadGoalFields[fieldName].GoType != "int64" || !threadGoalFields[fieldName].Required {
			t.Fatalf("ThreadGoal.%s field = %#v", fieldName, threadGoalFields[fieldName])
		}
	}
	if threadGoalFields["objective"].GoType != "string" || !threadGoalFields["objective"].Required ||
		threadGoalFields["status"].GoType != "ThreadGoalStatus" || !threadGoalFields["status"].Required ||
		threadGoalFields["threadId"].GoType != "string" || !threadGoalFields["threadId"].Required ||
		threadGoalFields["tokenBudget"].GoType != "*protocolv2.Nullable[int64]" {
		t.Fatalf("ThreadGoal fields = %#v", threadGoalFields)
	}
	threadGoalGetResponseFields := generatedFields("ThreadGoalGetResponse")
	if threadGoalGetResponseFields["goal"].GoType != "*protocolv2.Nullable[ThreadGoal]" ||
		threadGoalGetResponseFields["goal"].Required {
		t.Fatalf("ThreadGoalGetResponse.goal field = %#v", threadGoalGetResponseFields["goal"])
	}
	threadGoalSetResponseFields := generatedFields("ThreadGoalSetResponse")
	if threadGoalSetResponseFields["goal"].GoType != "ThreadGoal" || !threadGoalSetResponseFields["goal"].Required {
		t.Fatalf("ThreadGoalSetResponse.goal field = %#v", threadGoalSetResponseFields["goal"])
	}
	threadGoalUpdatedFields := generatedFields("ThreadGoalUpdatedNotification")
	if threadGoalUpdatedFields["goal"].GoType != "ThreadGoal" || !threadGoalUpdatedFields["goal"].Required ||
		threadGoalUpdatedFields["threadId"].GoType != "string" || !threadGoalUpdatedFields["threadId"].Required ||
		threadGoalUpdatedFields["turnId"].GoType != "*protocolv2.Nullable[string]" {
		t.Fatalf("ThreadGoalUpdatedNotification fields = %#v", threadGoalUpdatedFields)
	}
	threadMemoryModeSetFields := generatedFields("ThreadMemoryModeSetParams")
	if threadMemoryModeSetFields["mode"].GoType != "ThreadMemoryMode" || !threadMemoryModeSetFields["mode"].Required ||
		threadMemoryModeSetFields["threadId"].GoType != "string" || !threadMemoryModeSetFields["threadId"].Required {
		t.Fatalf("ThreadMemoryModeSetParams fields = %#v", threadMemoryModeSetFields)
	}
	threadNameUpdatedFields := generatedFields("ThreadNameUpdatedNotification")
	if threadNameUpdatedFields["threadId"].GoType != "string" || !threadNameUpdatedFields["threadId"].Required ||
		threadNameUpdatedFields["threadName"].GoType != "*protocolv2.Nullable[string]" {
		t.Fatalf("ThreadNameUpdatedNotification fields = %#v", threadNameUpdatedFields)
	}
	threadReadParamsFields := generatedFields("ThreadReadParams")
	if threadReadParamsFields["includeTurns"].GoType != "*bool" ||
		threadReadParamsFields["threadId"].GoType != "string" || !threadReadParamsFields["threadId"].Required {
		t.Fatalf("ThreadReadParams fields = %#v", threadReadParamsFields)
	}
	threadSetNameFields := generatedFields("ThreadSetNameParams")
	if threadSetNameFields["name"].GoType != "string" || !threadSetNameFields["name"].Required ||
		threadSetNameFields["threadId"].GoType != "string" || !threadSetNameFields["threadId"].Required {
		t.Fatalf("ThreadSetNameParams fields = %#v", threadSetNameFields)
	}
	threadShellCommandFields := generatedFields("ThreadShellCommandParams")
	if threadShellCommandFields["command"].GoType != "string" || !threadShellCommandFields["command"].Required ||
		threadShellCommandFields["threadId"].GoType != "string" || !threadShellCommandFields["threadId"].Required {
		t.Fatalf("ThreadShellCommandParams fields = %#v", threadShellCommandFields)
	}
	threadUnsubscribeResponseFields := generatedFields("ThreadUnsubscribeResponse")
	if threadUnsubscribeResponseFields["status"].GoType != "ThreadUnsubscribeStatus" || !threadUnsubscribeResponseFields["status"].Required {
		t.Fatalf("ThreadUnsubscribeResponse.status field = %#v", threadUnsubscribeResponseFields["status"])
	}
	for _, typeName := range []string{"AgentMessageDeltaNotification", "FileChangeOutputDeltaNotification", "PlanDeltaNotification"} {
		fields := generatedFields(typeName)
		for fieldName, wantGoType := range map[string]string{
			"delta":    "string",
			"itemId":   "string",
			"threadId": "string",
			"turnId":   "string",
		} {
			if fields[fieldName].GoType != wantGoType || !fields[fieldName].Required {
				t.Fatalf("%s.%s field = %#v", typeName, fieldName, fields[fieldName])
			}
		}
	}
	for _, typeName := range []string{"SkillsChangedNotification"} {
		if fields := selectedByName[typeName].Fields; len(fields) != 0 {
			t.Fatalf("%s fields = %#v, want empty notification payload", typeName, fields)
		}
	}
	externalAgentImportCompletedFields := generatedFields("ExternalAgentConfigImportCompletedNotification")
	if externalAgentImportCompletedFields["importId"].GoType != "string" ||
		!externalAgentImportCompletedFields["importId"].Required ||
		externalAgentImportCompletedFields["itemTypeResults"].GoType != "[]ExternalAgentConfigImportTypeResult" ||
		!externalAgentImportCompletedFields["itemTypeResults"].Required {
		t.Fatalf("ExternalAgentConfigImportCompletedNotification fields = %#v", externalAgentImportCompletedFields)
	}
	contextCompactedFields := generatedFields("ContextCompactedNotification")
	if contextCompactedFields["threadId"].GoType != "string" || !contextCompactedFields["threadId"].Required ||
		contextCompactedFields["turnId"].GoType != "string" || !contextCompactedFields["turnId"].Required {
		t.Fatalf("ContextCompactedNotification fields = %#v", contextCompactedFields)
	}
	deprecationNoticeFields := generatedFields("DeprecationNoticeNotification")
	if deprecationNoticeFields["details"].GoType != "*protocolv2.Nullable[string]" ||
		deprecationNoticeFields["summary"].GoType != "string" || !deprecationNoticeFields["summary"].Required {
		t.Fatalf("DeprecationNoticeNotification fields = %#v", deprecationNoticeFields)
	}
	guardianWarningFields := generatedFields("GuardianWarningNotification")
	if guardianWarningFields["message"].GoType != "string" || !guardianWarningFields["message"].Required ||
		guardianWarningFields["threadId"].GoType != "string" || !guardianWarningFields["threadId"].Required {
		t.Fatalf("GuardianWarningNotification fields = %#v", guardianWarningFields)
	}
	guardianReviewFields := generatedFields("GuardianApprovalReview")
	for fieldName, wantGoType := range map[string]string{
		"rationale":         "*protocolv2.Nullable[string]",
		"riskLevel":         "*protocolv2.Nullable[GuardianRiskLevel]",
		"status":            "GuardianApprovalReviewStatus",
		"userAuthorization": "*protocolv2.Nullable[GuardianUserAuthorization]",
	} {
		if guardianReviewFields[fieldName].GoType != wantGoType {
			t.Fatalf("GuardianApprovalReview.%s GoType = %q, want %q", fieldName, guardianReviewFields[fieldName].GoType, wantGoType)
		}
	}
	if !guardianReviewFields["status"].Required {
		t.Fatalf("GuardianApprovalReview.status should be required: %#v", guardianReviewFields["status"])
	}
	startedReviewFields := generatedFields("ItemGuardianApprovalReviewStartedNotification")
	for fieldName, wantGoType := range map[string]string{
		"action":       "GuardianApprovalReviewAction",
		"review":       "GuardianApprovalReview",
		"reviewId":     "string",
		"startedAtMs":  "int64",
		"targetItemId": "*protocolv2.Nullable[string]",
		"threadId":     "string",
		"turnId":       "string",
	} {
		if startedReviewFields[fieldName].GoType != wantGoType {
			t.Fatalf("ItemGuardianApprovalReviewStartedNotification.%s GoType = %q, want %q", fieldName, startedReviewFields[fieldName].GoType, wantGoType)
		}
	}
	for _, fieldName := range []string{"action", "review", "reviewId", "startedAtMs", "threadId", "turnId"} {
		if !startedReviewFields[fieldName].Required {
			t.Fatalf("ItemGuardianApprovalReviewStartedNotification.%s should be required: %#v", fieldName, startedReviewFields[fieldName])
		}
	}
	completedReviewFields := generatedFields("ItemGuardianApprovalReviewCompletedNotification")
	for fieldName, wantGoType := range map[string]string{
		"action":         "GuardianApprovalReviewAction",
		"completedAtMs":  "int64",
		"decisionSource": "AutoReviewDecisionSource",
		"review":         "GuardianApprovalReview",
		"reviewId":       "string",
		"startedAtMs":    "int64",
		"targetItemId":   "*protocolv2.Nullable[string]",
		"threadId":       "string",
		"turnId":         "string",
	} {
		if completedReviewFields[fieldName].GoType != wantGoType {
			t.Fatalf("ItemGuardianApprovalReviewCompletedNotification.%s GoType = %q, want %q", fieldName, completedReviewFields[fieldName].GoType, wantGoType)
		}
	}
	for _, fieldName := range []string{"action", "completedAtMs", "decisionSource", "review", "reviewId", "startedAtMs", "threadId", "turnId"} {
		if !completedReviewFields[fieldName].Required {
			t.Fatalf("ItemGuardianApprovalReviewCompletedNotification.%s should be required: %#v", fieldName, completedReviewFields[fieldName])
		}
	}
	mcpOauthCompletedFields := generatedFields("McpServerOauthLoginCompletedNotification")
	if mcpOauthCompletedFields["error"].GoType != "*protocolv2.Nullable[string]" ||
		mcpOauthCompletedFields["name"].GoType != "string" || !mcpOauthCompletedFields["name"].Required ||
		mcpOauthCompletedFields["success"].GoType != "bool" || !mcpOauthCompletedFields["success"].Required {
		t.Fatalf("McpServerOauthLoginCompletedNotification fields = %#v", mcpOauthCompletedFields)
	}
	mcpStatusUpdatedFields := generatedFields("McpServerStatusUpdatedNotification")
	if mcpStatusUpdatedFields["error"].GoType != "*protocolv2.Nullable[string]" ||
		mcpStatusUpdatedFields["name"].GoType != "string" || !mcpStatusUpdatedFields["name"].Required ||
		mcpStatusUpdatedFields["status"].GoType != "McpServerStartupState" || !mcpStatusUpdatedFields["status"].Required {
		t.Fatalf("McpServerStatusUpdatedNotification fields = %#v", mcpStatusUpdatedFields)
	}
	mcpProgressFields := generatedFields("McpToolCallProgressNotification")
	for fieldName, wantGoType := range map[string]string{"itemId": "string", "message": "string", "threadId": "string", "turnId": "string"} {
		if mcpProgressFields[fieldName].GoType != wantGoType || !mcpProgressFields[fieldName].Required {
			t.Fatalf("McpToolCallProgressNotification.%s field = %#v", fieldName, mcpProgressFields[fieldName])
		}
	}
	mcpResourceReadFields := generatedFields("McpResourceReadParams")
	for fieldName, wantGoType := range map[string]string{"server": "string", "uri": "string"} {
		if mcpResourceReadFields[fieldName].GoType != wantGoType || !mcpResourceReadFields[fieldName].Required {
			t.Fatalf("McpResourceReadParams.%s field = %#v", fieldName, mcpResourceReadFields[fieldName])
		}
	}
	if mcpResourceReadFields["threadId"].GoType != "*protocolv2.Nullable[string]" {
		t.Fatalf("McpResourceReadParams.threadId field = %#v", mcpResourceReadFields["threadId"])
	}
	mcpElicitationFields := generatedFields("McpServerElicitationRequestParams")
	for fieldName, wantGoType := range map[string]string{"serverName": "string", "threadId": "string"} {
		if mcpElicitationFields[fieldName].GoType != wantGoType || !mcpElicitationFields[fieldName].Required {
			t.Fatalf("McpServerElicitationRequestParams.%s field = %#v", fieldName, mcpElicitationFields[fieldName])
		}
	}
	if mcpElicitationFields["turnId"].GoType != "*protocolv2.Nullable[string]" {
		t.Fatalf("McpServerElicitationRequestParams.turnId field = %#v", mcpElicitationFields["turnId"])
	}
	mcpElicitationResponseFields := generatedFields("McpServerElicitationRequestResponse")
	if mcpElicitationResponseFields["action"].GoType != "McpServerElicitationAction" ||
		!mcpElicitationResponseFields["action"].Required ||
		mcpElicitationResponseFields["_meta"].GoType != "*protocolv2.JSONValue" ||
		mcpElicitationResponseFields["_meta"].Kind != FieldPlanJSONValue ||
		mcpElicitationResponseFields["content"].GoType != "*protocolv2.JSONValue" ||
		mcpElicitationResponseFields["content"].Kind != FieldPlanJSONValue {
		t.Fatalf("McpServerElicitationRequestResponse fields = %#v", mcpElicitationResponseFields)
	}
	mcpOauthLoginFields := generatedFields("McpServerOauthLoginParams")
	if mcpOauthLoginFields["name"].GoType != "string" || !mcpOauthLoginFields["name"].Required ||
		mcpOauthLoginFields["scopes"].GoType != "*protocolv2.Nullable[[]string]" ||
		mcpOauthLoginFields["timeoutSecs"].GoType != "*protocolv2.Nullable[int64]" {
		t.Fatalf("McpServerOauthLoginParams fields = %#v", mcpOauthLoginFields)
	}
	mcpOauthLoginResponseFields := generatedFields("McpServerOauthLoginResponse")
	if mcpOauthLoginResponseFields["authorizationUrl"].GoType != "string" || !mcpOauthLoginResponseFields["authorizationUrl"].Required {
		t.Fatalf("McpServerOauthLoginResponse.authorizationUrl field = %#v", mcpOauthLoginResponseFields["authorizationUrl"])
	}
	if fields := selectedByName["McpServerRefreshResponse"].Fields; len(fields) != 0 {
		t.Fatalf("McpServerRefreshResponse fields = %#v, want empty response payload", fields)
	}
	mcpToolCallFields := generatedFields("McpServerToolCallParams")
	for fieldName, wantGoType := range map[string]string{"server": "string", "threadId": "string", "tool": "string"} {
		if mcpToolCallFields[fieldName].GoType != wantGoType || !mcpToolCallFields[fieldName].Required {
			t.Fatalf("McpServerToolCallParams.%s field = %#v", fieldName, mcpToolCallFields[fieldName])
		}
	}
	if mcpToolCallFields["_meta"].GoType != "*protocolv2.JSONValue" || mcpToolCallFields["_meta"].Kind != FieldPlanJSONValue ||
		mcpToolCallFields["arguments"].GoType != "*protocolv2.JSONValue" || mcpToolCallFields["arguments"].Kind != FieldPlanJSONValue {
		t.Fatalf("McpServerToolCallParams JSONValue fields = %#v", mcpToolCallFields)
	}
	mcpToolCallResponseFields := generatedFields("McpServerToolCallResponse")
	if mcpToolCallResponseFields["content"].GoType != "[]protocolv2.JSONValue" ||
		mcpToolCallResponseFields["content"].Kind != FieldPlanArrayJSONValue ||
		!mcpToolCallResponseFields["content"].Required ||
		mcpToolCallResponseFields["isError"].GoType != "*protocolv2.Nullable[bool]" ||
		mcpToolCallResponseFields["_meta"].GoType != "*protocolv2.JSONValue" ||
		mcpToolCallResponseFields["_meta"].Kind != FieldPlanJSONValue ||
		mcpToolCallResponseFields["structuredContent"].GoType != "*protocolv2.JSONValue" ||
		mcpToolCallResponseFields["structuredContent"].Kind != FieldPlanJSONValue {
		t.Fatalf("McpServerToolCallResponse fields = %#v", mcpToolCallResponseFields)
	}
	mcpResourceReadResponseFields := generatedFields("McpResourceReadResponse")
	if mcpResourceReadResponseFields["contents"].GoType != "[]ResourceContent" ||
		mcpResourceReadResponseFields["contents"].Kind != FieldPlanArrayRef ||
		!mcpResourceReadResponseFields["contents"].Required {
		t.Fatalf("McpResourceReadResponse fields = %#v", mcpResourceReadResponseFields)
	}
	mcpStatusListFields := generatedFields("ListMcpServerStatusResponse")
	if mcpStatusListFields["data"].GoType != "[]McpServerStatus" ||
		mcpStatusListFields["data"].Kind != FieldPlanArrayRef ||
		!mcpStatusListFields["data"].Required ||
		mcpStatusListFields["nextCursor"].GoType != "*protocolv2.Nullable[string]" {
		t.Fatalf("ListMcpServerStatusResponse fields = %#v", mcpStatusListFields)
	}
	mcpServerStatusFields := generatedFields("McpServerStatus")
	if mcpServerStatusFields["authStatus"].GoType != "McpAuthStatus" ||
		!mcpServerStatusFields["authStatus"].Required ||
		mcpServerStatusFields["resourceTemplates"].GoType != "[]ResourceTemplate" ||
		!mcpServerStatusFields["resourceTemplates"].Required ||
		mcpServerStatusFields["resources"].GoType != "[]Resource" ||
		!mcpServerStatusFields["resources"].Required ||
		mcpServerStatusFields["tools"].GoType != "map[string]Tool" ||
		mcpServerStatusFields["tools"].Kind != FieldPlanTypedMap ||
		!mcpServerStatusFields["tools"].Required {
		t.Fatalf("McpServerStatus fields = %#v", mcpServerStatusFields)
	}
	resourceFields := generatedFields("Resource")
	if resourceFields["_meta"].GoType != "*protocolv2.JSONValue" ||
		resourceFields["_meta"].Kind != FieldPlanJSONValue ||
		resourceFields["annotations"].GoType != "*protocolv2.JSONValue" ||
		resourceFields["annotations"].Kind != FieldPlanJSONValue ||
		resourceFields["icons"].GoType != "*protocolv2.Nullable[[]protocolv2.JSONValue]" ||
		resourceFields["icons"].Kind != FieldPlanArrayJSONValue ||
		resourceFields["size"].GoType != "*protocolv2.Nullable[int64]" ||
		resourceFields["name"].GoType != "string" || !resourceFields["name"].Required ||
		resourceFields["uri"].GoType != "string" || !resourceFields["uri"].Required {
		t.Fatalf("Resource fields = %#v", resourceFields)
	}
	resourceTemplateFields := generatedFields("ResourceTemplate")
	if resourceTemplateFields["annotations"].GoType != "*protocolv2.JSONValue" ||
		resourceTemplateFields["annotations"].Kind != FieldPlanJSONValue ||
		resourceTemplateFields["name"].GoType != "string" || !resourceTemplateFields["name"].Required ||
		resourceTemplateFields["uriTemplate"].GoType != "string" || !resourceTemplateFields["uriTemplate"].Required {
		t.Fatalf("ResourceTemplate fields = %#v", resourceTemplateFields)
	}
	toolFields := generatedFields("Tool")
	if toolFields["_meta"].GoType != "*protocolv2.JSONValue" ||
		toolFields["_meta"].Kind != FieldPlanJSONValue ||
		toolFields["annotations"].GoType != "*protocolv2.JSONValue" ||
		toolFields["annotations"].Kind != FieldPlanJSONValue ||
		toolFields["icons"].GoType != "*protocolv2.Nullable[[]protocolv2.JSONValue]" ||
		toolFields["icons"].Kind != FieldPlanArrayJSONValue ||
		toolFields["inputSchema"].GoType != "protocolv2.JSONValue" ||
		toolFields["inputSchema"].Kind != FieldPlanJSONValue ||
		!toolFields["inputSchema"].Required ||
		toolFields["outputSchema"].GoType != "*protocolv2.JSONValue" ||
		toolFields["outputSchema"].Kind != FieldPlanJSONValue ||
		toolFields["name"].GoType != "string" || !toolFields["name"].Required {
		t.Fatalf("Tool fields = %#v", toolFields)
	}
	pluginShareDeleteFields := generatedFields("PluginShareDeleteParams")
	if pluginShareDeleteFields["remotePluginId"].GoType != "string" || !pluginShareDeleteFields["remotePluginId"].Required {
		t.Fatalf("PluginShareDeleteParams.remotePluginId field = %#v", pluginShareDeleteFields["remotePluginId"])
	}
	for _, typeName := range []string{"PluginShareDeleteResponse", "PluginShareListParams", "PluginUninstallResponse"} {
		if fields := selectedByName[typeName].Fields; len(fields) != 0 {
			t.Fatalf("%s fields = %#v, want empty plugin payload", typeName, fields)
		}
	}
	pluginShareSaveResponseFields := generatedFields("PluginShareSaveResponse")
	for fieldName, wantGoType := range map[string]string{"remotePluginId": "string", "shareUrl": "string"} {
		if pluginShareSaveResponseFields[fieldName].GoType != wantGoType || !pluginShareSaveResponseFields[fieldName].Required {
			t.Fatalf("PluginShareSaveResponse.%s field = %#v", fieldName, pluginShareSaveResponseFields[fieldName])
		}
	}
	pluginInstallResponseFields := generatedFields("PluginInstallResponse")
	if pluginInstallResponseFields["appsNeedingAuth"].GoType != "[]AppSummary" ||
		pluginInstallResponseFields["appsNeedingAuth"].Kind != FieldPlanArrayRef ||
		!pluginInstallResponseFields["appsNeedingAuth"].Required ||
		pluginInstallResponseFields["authPolicy"].GoType != "PluginAuthPolicy" ||
		!pluginInstallResponseFields["authPolicy"].Required {
		t.Fatalf("PluginInstallResponse fields = %#v", pluginInstallResponseFields)
	}
	pluginListResponseFields := generatedFields("PluginListResponse")
	if pluginListResponseFields["featuredPluginIds"].GoType != "*[]string" ||
		pluginListResponseFields["marketplaceLoadErrors"].GoType != "*[]MarketplaceLoadErrorInfo" ||
		pluginListResponseFields["marketplaces"].GoType != "[]PluginMarketplaceEntry" ||
		pluginListResponseFields["marketplaces"].Kind != FieldPlanArrayRef ||
		!pluginListResponseFields["marketplaces"].Required {
		t.Fatalf("PluginListResponse fields = %#v", pluginListResponseFields)
	}
	pluginMarketplaceEntryFields := generatedFields("PluginMarketplaceEntry")
	if pluginMarketplaceEntryFields["interface"].GoType != "*protocolv2.Nullable[MarketplaceInterface]" ||
		pluginMarketplaceEntryFields["path"].GoType != "*protocolv2.Nullable[string]" ||
		pluginMarketplaceEntryFields["plugins"].GoType != "[]PluginSummary" ||
		!pluginMarketplaceEntryFields["plugins"].Required {
		t.Fatalf("PluginMarketplaceEntry fields = %#v", pluginMarketplaceEntryFields)
	}
	pluginSummaryFields := generatedFields("PluginSummary")
	if pluginSummaryFields["availability"].GoType != "*PluginAvailability" ||
		pluginSummaryFields["availability"].Kind != FieldPlanAllOfRef ||
		pluginSummaryFields["interface"].GoType != "*protocolv2.Nullable[PluginInterface]" ||
		pluginSummaryFields["keywords"].GoType != "*[]string" ||
		pluginSummaryFields["shareContext"].GoType != "*protocolv2.Nullable[PluginShareContext]" ||
		pluginSummaryFields["source"].GoType != "PluginSource" ||
		!pluginSummaryFields["source"].Required {
		t.Fatalf("PluginSummary fields = %#v", pluginSummaryFields)
	}
	pluginReadResponseFields := generatedFields("PluginReadResponse")
	if pluginReadResponseFields["plugin"].GoType != "PluginDetail" || !pluginReadResponseFields["plugin"].Required {
		t.Fatalf("PluginReadResponse.plugin field = %#v", pluginReadResponseFields["plugin"])
	}
	pluginDetailFields := generatedFields("PluginDetail")
	if pluginDetailFields["apps"].GoType != "[]AppSummary" ||
		!pluginDetailFields["apps"].Required ||
		pluginDetailFields["hooks"].GoType != "[]PluginHookSummary" ||
		!pluginDetailFields["hooks"].Required ||
		pluginDetailFields["marketplacePath"].GoType != "*protocolv2.Nullable[string]" ||
		pluginDetailFields["mcpServers"].GoType != "[]string" ||
		!pluginDetailFields["mcpServers"].Required ||
		pluginDetailFields["shareUrl"].GoType != "*protocolv2.Nullable[string]" ||
		pluginDetailFields["shareUrl"].Required ||
		pluginDetailFields["skills"].GoType != "[]SkillSummary" ||
		!pluginDetailFields["skills"].Required ||
		pluginDetailFields["summary"].GoType != "PluginSummary" ||
		!pluginDetailFields["summary"].Required {
		t.Fatalf("PluginDetail fields = %#v", pluginDetailFields)
	}
	pluginShareListResponseFields := generatedFields("PluginShareListResponse")
	if pluginShareListResponseFields["data"].GoType != "[]PluginShareListItem" ||
		pluginShareListResponseFields["data"].Kind != FieldPlanArrayRef ||
		!pluginShareListResponseFields["data"].Required {
		t.Fatalf("PluginShareListResponse fields = %#v", pluginShareListResponseFields)
	}
	pluginShareListItemFields := generatedFields("PluginShareListItem")
	if pluginShareListItemFields["localPluginPath"].GoType != "*protocolv2.Nullable[string]" ||
		pluginShareListItemFields["plugin"].GoType != "PluginSummary" ||
		!pluginShareListItemFields["plugin"].Required {
		t.Fatalf("PluginShareListItem fields = %#v", pluginShareListItemFields)
	}
	pluginShareSaveParamsFields := generatedFields("PluginShareSaveParams")
	if pluginShareSaveParamsFields["discoverability"].GoType != "*protocolv2.Nullable[PluginShareDiscoverability]" ||
		pluginShareSaveParamsFields["pluginPath"].GoType != "string" ||
		!pluginShareSaveParamsFields["pluginPath"].Required ||
		pluginShareSaveParamsFields["remotePluginId"].GoType != "*protocolv2.Nullable[string]" ||
		pluginShareSaveParamsFields["shareTargets"].GoType != "*protocolv2.Nullable[[]PluginShareTarget]" {
		t.Fatalf("PluginShareSaveParams fields = %#v", pluginShareSaveParamsFields)
	}
	pluginShareTargetFields := generatedFields("PluginShareTarget")
	if pluginShareTargetFields["principalId"].GoType != "string" ||
		!pluginShareTargetFields["principalId"].Required ||
		pluginShareTargetFields["principalType"].GoType != "PluginSharePrincipalType" ||
		!pluginShareTargetFields["principalType"].Required {
		t.Fatalf("PluginShareTarget fields = %#v", pluginShareTargetFields)
	}
	pluginShareUpdateTargetsParamsFields := generatedFields("PluginShareUpdateTargetsParams")
	if pluginShareUpdateTargetsParamsFields["discoverability"].GoType != "PluginShareUpdateDiscoverability" ||
		!pluginShareUpdateTargetsParamsFields["discoverability"].Required ||
		pluginShareUpdateTargetsParamsFields["remotePluginId"].GoType != "string" ||
		!pluginShareUpdateTargetsParamsFields["remotePluginId"].Required ||
		pluginShareUpdateTargetsParamsFields["shareTargets"].GoType != "[]PluginShareTarget" ||
		!pluginShareUpdateTargetsParamsFields["shareTargets"].Required {
		t.Fatalf("PluginShareUpdateTargetsParams fields = %#v", pluginShareUpdateTargetsParamsFields)
	}
	pluginShareUpdateTargetsResponseFields := generatedFields("PluginShareUpdateTargetsResponse")
	if pluginShareUpdateTargetsResponseFields["discoverability"].GoType != "PluginShareDiscoverability" ||
		!pluginShareUpdateTargetsResponseFields["discoverability"].Required ||
		pluginShareUpdateTargetsResponseFields["principals"].GoType != "[]PluginSharePrincipal" ||
		!pluginShareUpdateTargetsResponseFields["principals"].Required {
		t.Fatalf("PluginShareUpdateTargetsResponse fields = %#v", pluginShareUpdateTargetsResponseFields)
	}
	pluginSkillReadFields := generatedFields("PluginSkillReadParams")
	for fieldName, wantGoType := range map[string]string{"remoteMarketplaceName": "string", "remotePluginId": "string", "skillName": "string"} {
		if pluginSkillReadFields[fieldName].GoType != wantGoType || !pluginSkillReadFields[fieldName].Required {
			t.Fatalf("PluginSkillReadParams.%s field = %#v", fieldName, pluginSkillReadFields[fieldName])
		}
	}
	pluginSkillReadResponseFields := generatedFields("PluginSkillReadResponse")
	if pluginSkillReadResponseFields["contents"].GoType != "*protocolv2.Nullable[string]" {
		t.Fatalf("PluginSkillReadResponse.contents field = %#v", pluginSkillReadResponseFields["contents"])
	}
	pluginUninstallFields := generatedFields("PluginUninstallParams")
	if pluginUninstallFields["pluginId"].GoType != "string" || !pluginUninstallFields["pluginId"].Required {
		t.Fatalf("PluginUninstallParams.pluginId field = %#v", pluginUninstallFields["pluginId"])
	}
	for _, typeName := range []string{"ExperimentalFeatureEnablementSetParams", "ExperimentalFeatureEnablementSetResponse"} {
		fields := generatedFields(typeName)
		if fields["enablement"].GoType != "map[string]bool" ||
			fields["enablement"].Kind != FieldPlanTypedMap ||
			!fields["enablement"].Required {
			t.Fatalf("%s.enablement field = %#v", typeName, fields["enablement"])
		}
	}
	experimentalFeatureListFields := generatedFields("ExperimentalFeatureListParams")
	if experimentalFeatureListFields["cursor"].GoType != "*protocolv2.Nullable[string]" ||
		experimentalFeatureListFields["limit"].GoType != "*protocolv2.Nullable[uint32]" {
		t.Fatalf("ExperimentalFeatureListParams fields = %#v", experimentalFeatureListFields)
	}
	externalDetectFields := generatedFields("ExternalAgentConfigDetectParams")
	if externalDetectFields["cwds"].GoType != "*protocolv2.Nullable[[]string]" ||
		externalDetectFields["includeHome"].GoType != "*bool" {
		t.Fatalf("ExternalAgentConfigDetectParams fields = %#v", externalDetectFields)
	}
	externalDetectResponseFields := generatedFields("ExternalAgentConfigDetectResponse")
	if externalDetectResponseFields["items"].GoType != "[]ExternalAgentConfigMigrationItem" ||
		externalDetectResponseFields["items"].Kind != FieldPlanArrayRef ||
		!externalDetectResponseFields["items"].Required {
		t.Fatalf("ExternalAgentConfigDetectResponse.items field = %#v", externalDetectResponseFields["items"])
	}
	externalImportFields := generatedFields("ExternalAgentConfigImportParams")
	if externalImportFields["migrationItems"].GoType != "[]ExternalAgentConfigMigrationItem" ||
		externalImportFields["migrationItems"].Kind != FieldPlanArrayRef ||
		!externalImportFields["migrationItems"].Required {
		t.Fatalf("ExternalAgentConfigImportParams.migrationItems field = %#v", externalImportFields["migrationItems"])
	}
	migrationItemFields := generatedFields("ExternalAgentConfigMigrationItem")
	if migrationItemFields["cwd"].GoType != "*protocolv2.Nullable[string]" ||
		migrationItemFields["details"].GoType != "*protocolv2.Nullable[MigrationDetails]" ||
		migrationItemFields["description"].GoType != "string" ||
		migrationItemFields["itemType"].GoType != "ExternalAgentConfigMigrationItemType" {
		t.Fatalf("ExternalAgentConfigMigrationItem fields = %#v", migrationItemFields)
	}
	migrationDetailsFields := generatedFields("MigrationDetails")
	for fieldName, wantGoType := range map[string]string{
		"commands":   "*[]CommandMigration",
		"hooks":      "*[]HookMigration",
		"mcpServers": "*[]McpServerMigration",
		"plugins":    "*[]PluginsMigration",
		"sessions":   "*[]SessionMigration",
		"subagents":  "*[]SubagentMigration",
	} {
		if migrationDetailsFields[fieldName].GoType != wantGoType || migrationDetailsFields[fieldName].Kind != FieldPlanArrayRef {
			t.Fatalf("MigrationDetails.%s field = %#v", fieldName, migrationDetailsFields[fieldName])
		}
	}
	externalAgentConfigImportResponseFields := generatedFields("ExternalAgentConfigImportResponse")
	if externalAgentConfigImportResponseFields["importId"].GoType != "string" ||
		!externalAgentConfigImportResponseFields["importId"].Required {
		t.Fatalf("ExternalAgentConfigImportResponse.importId field = %#v", externalAgentConfigImportResponseFields["importId"])
	}
	for _, typeName := range []string{"ThreadApproveGuardianDeniedActionResponse", "TurnInterruptResponse"} {
		if fields := selectedByName[typeName].Fields; len(fields) != 0 {
			t.Fatalf("%s fields = %#v, want empty payload", typeName, fields)
		}
	}
	hooksListFields := generatedFields("HooksListParams")
	if hooksListFields["cwds"].GoType != "*[]string" {
		t.Fatalf("HooksListParams.cwds field = %#v", hooksListFields["cwds"])
	}
	hooksListResponseFields := generatedFields("HooksListResponse")
	if hooksListResponseFields["data"].GoType != "[]HooksListEntry" ||
		hooksListResponseFields["data"].Kind != FieldPlanArrayRef ||
		!hooksListResponseFields["data"].Required {
		t.Fatalf("HooksListResponse.data field = %#v", hooksListResponseFields["data"])
	}
	hooksListEntryFields := generatedFields("HooksListEntry")
	for fieldName, wantGoType := range map[string]string{
		"cwd":      "string",
		"errors":   "[]HookErrorInfo",
		"hooks":    "[]HookMetadata",
		"warnings": "[]string",
	} {
		if hooksListEntryFields[fieldName].GoType != wantGoType || !hooksListEntryFields[fieldName].Required {
			t.Fatalf("HooksListEntry.%s field = %#v", fieldName, hooksListEntryFields[fieldName])
		}
	}
	if hooksListEntryFields["errors"].Kind != FieldPlanArrayRef ||
		hooksListEntryFields["hooks"].Kind != FieldPlanArrayRef ||
		hooksListEntryFields["warnings"].Kind != FieldPlanArrayString {
		t.Fatalf("HooksListEntry array field kinds = %#v", hooksListEntryFields)
	}
	hookErrorFields := generatedFields("HookErrorInfo")
	for fieldName := range map[string]struct{}{"message": {}, "path": {}} {
		if hookErrorFields[fieldName].GoType != "string" || !hookErrorFields[fieldName].Required {
			t.Fatalf("HookErrorInfo.%s field = %#v", fieldName, hookErrorFields[fieldName])
		}
	}
	hookMetadataFields := generatedFields("HookMetadata")
	for fieldName, wantGoType := range map[string]string{
		"command":       "*protocolv2.Nullable[string]",
		"currentHash":   "string",
		"displayOrder":  "int64",
		"enabled":       "bool",
		"eventName":     "HookEventName",
		"handlerType":   "HookHandlerType",
		"isManaged":     "bool",
		"key":           "string",
		"matcher":       "*protocolv2.Nullable[string]",
		"pluginId":      "*protocolv2.Nullable[string]",
		"source":        "HookSource",
		"sourcePath":    "string",
		"statusMessage": "*protocolv2.Nullable[string]",
		"timeoutSec":    "uint64",
		"trustStatus":   "HookTrustStatus",
	} {
		if hookMetadataFields[fieldName].GoType != wantGoType {
			t.Fatalf("HookMetadata.%s GoType = %q, want %q", fieldName, hookMetadataFields[fieldName].GoType, wantGoType)
		}
	}
	for _, fieldName := range []string{"currentHash", "displayOrder", "enabled", "eventName", "handlerType", "isManaged", "key", "source", "sourcePath", "timeoutSec", "trustStatus"} {
		if !hookMetadataFields[fieldName].Required {
			t.Fatalf("HookMetadata.%s field should be required: %#v", fieldName, hookMetadataFields[fieldName])
		}
	}
	hookOutputEntryFields := generatedFields("HookOutputEntry")
	if hookOutputEntryFields["kind"].GoType != "HookOutputEntryKind" ||
		!hookOutputEntryFields["kind"].Required ||
		hookOutputEntryFields["text"].GoType != "string" ||
		!hookOutputEntryFields["text"].Required {
		t.Fatalf("HookOutputEntry fields = %#v", hookOutputEntryFields)
	}
	hookRunSummaryFields := generatedFields("HookRunSummary")
	for fieldName, wantGoType := range map[string]string{
		"completedAt":   "*protocolv2.Nullable[int64]",
		"displayOrder":  "int64",
		"durationMs":    "*protocolv2.Nullable[int64]",
		"entries":       "[]HookOutputEntry",
		"eventName":     "HookEventName",
		"executionMode": "HookExecutionMode",
		"handlerType":   "HookHandlerType",
		"id":            "string",
		"scope":         "HookScope",
		"source":        "*HookSource",
		"sourcePath":    "string",
		"startedAt":     "int64",
		"status":        "HookRunStatus",
		"statusMessage": "*protocolv2.Nullable[string]",
	} {
		if hookRunSummaryFields[fieldName].GoType != wantGoType {
			t.Fatalf("HookRunSummary.%s GoType = %q, want %q", fieldName, hookRunSummaryFields[fieldName].GoType, wantGoType)
		}
	}
	for _, fieldName := range []string{"displayOrder", "entries", "eventName", "executionMode", "handlerType", "id", "scope", "sourcePath", "startedAt", "status"} {
		if !hookRunSummaryFields[fieldName].Required {
			t.Fatalf("HookRunSummary.%s field should be required: %#v", fieldName, hookRunSummaryFields[fieldName])
		}
	}
	for _, typeName := range []string{"HookCompletedNotification", "HookStartedNotification"} {
		fields := generatedFields(typeName)
		if fields["run"].GoType != "HookRunSummary" || !fields["run"].Required ||
			fields["threadId"].GoType != "string" || !fields["threadId"].Required ||
			fields["turnId"].GoType != "*protocolv2.Nullable[string]" {
			t.Fatalf("%s fields = %#v", typeName, fields)
		}
	}
	skillsConfigWriteResponseFields := generatedFields("SkillsConfigWriteResponse")
	if skillsConfigWriteResponseFields["effectiveEnabled"].GoType != "bool" ||
		!skillsConfigWriteResponseFields["effectiveEnabled"].Required {
		t.Fatalf("SkillsConfigWriteResponse.effectiveEnabled field = %#v", skillsConfigWriteResponseFields["effectiveEnabled"])
	}
	skillsListFields := generatedFields("SkillsListParams")
	if skillsListFields["cwds"].GoType != "*[]string" || skillsListFields["forceReload"].GoType != "*bool" {
		t.Fatalf("SkillsListParams fields = %#v", skillsListFields)
	}
	reviewStartParamsFields := generatedFields("ReviewStartParams")
	if reviewStartParamsFields["delivery"].GoType != "*protocolv2.Nullable[ReviewDelivery]" ||
		reviewStartParamsFields["target"].GoType != "ReviewTarget" ||
		!reviewStartParamsFields["target"].Required ||
		reviewStartParamsFields["threadId"].GoType != "string" ||
		!reviewStartParamsFields["threadId"].Required {
		t.Fatalf("ReviewStartParams fields = %#v", reviewStartParamsFields)
	}
	skillsListResponseFields := generatedFields("SkillsListResponse")
	if skillsListResponseFields["data"].GoType != "[]SkillsListEntry" ||
		skillsListResponseFields["data"].Kind != FieldPlanArrayRef ||
		!skillsListResponseFields["data"].Required {
		t.Fatalf("SkillsListResponse fields = %#v", skillsListResponseFields)
	}
	skillsListEntryFields := generatedFields("SkillsListEntry")
	if skillsListEntryFields["cwd"].GoType != "string" ||
		!skillsListEntryFields["cwd"].Required ||
		skillsListEntryFields["errors"].GoType != "[]SkillErrorInfo" ||
		!skillsListEntryFields["errors"].Required ||
		skillsListEntryFields["skills"].GoType != "[]SkillMetadata" ||
		!skillsListEntryFields["skills"].Required {
		t.Fatalf("SkillsListEntry fields = %#v", skillsListEntryFields)
	}
	skillMetadataFields := generatedFields("SkillMetadata")
	if skillMetadataFields["dependencies"].GoType != "*protocolv2.Nullable[SkillDependencies]" ||
		skillMetadataFields["description"].GoType != "string" ||
		!skillMetadataFields["description"].Required ||
		skillMetadataFields["interface"].GoType != "*protocolv2.Nullable[SkillInterface]" ||
		skillMetadataFields["path"].GoType != "string" ||
		!skillMetadataFields["path"].Required ||
		skillMetadataFields["scope"].GoType != "SkillScope" ||
		!skillMetadataFields["scope"].Required ||
		skillMetadataFields["shortDescription"].GoType != "*protocolv2.Nullable[string]" {
		t.Fatalf("SkillMetadata fields = %#v", skillMetadataFields)
	}
	skillDependenciesFields := generatedFields("SkillDependencies")
	if skillDependenciesFields["tools"].GoType != "[]SkillToolDependency" ||
		!skillDependenciesFields["tools"].Required {
		t.Fatalf("SkillDependencies fields = %#v", skillDependenciesFields)
	}
	skillToolDependencyFields := generatedFields("SkillToolDependency")
	for fieldName, wantGoType := range map[string]string{
		"command":     "*protocolv2.Nullable[string]",
		"description": "*protocolv2.Nullable[string]",
		"transport":   "*protocolv2.Nullable[string]",
		"type":        "string",
		"url":         "*protocolv2.Nullable[string]",
		"value":       "string",
	} {
		if skillToolDependencyFields[fieldName].GoType != wantGoType {
			t.Fatalf("SkillToolDependency.%s field = %#v", fieldName, skillToolDependencyFields[fieldName])
		}
	}
	for _, fieldName := range []string{"type", "value"} {
		if !skillToolDependencyFields[fieldName].Required {
			t.Fatalf("SkillToolDependency.%s field should be required: %#v", fieldName, skillToolDependencyFields[fieldName])
		}
	}
	reasoningSummaryPartFields := generatedFields("ReasoningSummaryPartAddedNotification")
	for fieldName, wantGoType := range map[string]string{"itemId": "string", "summaryIndex": "int64", "threadId": "string", "turnId": "string"} {
		if reasoningSummaryPartFields[fieldName].GoType != wantGoType || !reasoningSummaryPartFields[fieldName].Required {
			t.Fatalf("ReasoningSummaryPartAddedNotification.%s field = %#v", fieldName, reasoningSummaryPartFields[fieldName])
		}
	}
	reasoningSummaryDeltaFields := generatedFields("ReasoningSummaryTextDeltaNotification")
	for fieldName, wantGoType := range map[string]string{"delta": "string", "itemId": "string", "summaryIndex": "int64", "threadId": "string", "turnId": "string"} {
		if reasoningSummaryDeltaFields[fieldName].GoType != wantGoType || !reasoningSummaryDeltaFields[fieldName].Required {
			t.Fatalf("ReasoningSummaryTextDeltaNotification.%s field = %#v", fieldName, reasoningSummaryDeltaFields[fieldName])
		}
	}
	reasoningTextDeltaFields := generatedFields("ReasoningTextDeltaNotification")
	for fieldName, wantGoType := range map[string]string{"contentIndex": "int64", "delta": "string", "itemId": "string", "threadId": "string", "turnId": "string"} {
		if reasoningTextDeltaFields[fieldName].GoType != wantGoType || !reasoningTextDeltaFields[fieldName].Required {
			t.Fatalf("ReasoningTextDeltaNotification.%s field = %#v", fieldName, reasoningTextDeltaFields[fieldName])
		}
	}
	remoteStatusFields := generatedFields("RemoteControlStatusChangedNotification")
	if remoteStatusFields["environmentId"].GoType != "*protocolv2.Nullable[string]" ||
		remoteStatusFields["status"].GoType != "RemoteControlConnectionStatus" || !remoteStatusFields["status"].Required {
		t.Fatalf("RemoteControlStatusChangedNotification fields = %#v", remoteStatusFields)
	}
	terminalInteractionFields := generatedFields("TerminalInteractionNotification")
	for fieldName, wantGoType := range map[string]string{"itemId": "string", "processId": "string", "stdin": "string", "threadId": "string", "turnId": "string"} {
		if terminalInteractionFields[fieldName].GoType != wantGoType || !terminalInteractionFields[fieldName].Required {
			t.Fatalf("TerminalInteractionNotification.%s field = %#v", fieldName, terminalInteractionFields[fieldName])
		}
	}
	turnDiffUpdatedFields := generatedFields("TurnDiffUpdatedNotification")
	for fieldName, wantGoType := range map[string]string{"diff": "string", "threadId": "string", "turnId": "string"} {
		if turnDiffUpdatedFields[fieldName].GoType != wantGoType || !turnDiffUpdatedFields[fieldName].Required {
			t.Fatalf("TurnDiffUpdatedNotification.%s field = %#v", fieldName, turnDiffUpdatedFields[fieldName])
		}
	}
	warningFields := generatedFields("WarningNotification")
	if warningFields["message"].GoType != "string" || !warningFields["message"].Required ||
		warningFields["threadId"].GoType != "*protocolv2.Nullable[string]" {
		t.Fatalf("WarningNotification fields = %#v", warningFields)
	}
	for _, typeName := range []string{"ThreadStartResponse", "ThreadResumeResponse", "ThreadForkResponse"} {
		response := selectedByName[typeName]
		fields := map[string]FieldPlan{}
		for _, field := range response.Fields {
			fields[field.FieldName] = field
		}
		for fieldName, wantGoType := range map[string]string{
			"activePermissionProfile": "*protocolv2.Nullable[ActivePermissionProfile]",
			"approvalPolicy":          "AskForApproval",
			"approvalsReviewer":       "ApprovalsReviewer",
			"cwd":                     "string",
			"instructionSources":      "*[]string",
			"model":                   "string",
			"modelProvider":           "string",
			"reasoningEffort":         "*protocolv2.Nullable[ReasoningEffort]",
			"sandbox":                 "SandboxPolicy",
			"serviceTier":             "*protocolv2.Nullable[string]",
			"thread":                  "Thread",
		} {
			if got := fields[fieldName].GoType; got != wantGoType {
				t.Fatalf("%s.%s GoType = %q, want %q", typeName, fieldName, got, wantGoType)
			}
		}
		for _, fieldName := range []string{"approvalPolicy", "approvalsReviewer", "cwd", "model", "modelProvider", "sandbox", "thread"} {
			if !fields[fieldName].Required {
				t.Fatalf("%s.%s must remain required", typeName, fieldName)
			}
		}
		if fields["serviceTier"].Kind != FieldPlanNullableServiceTier {
			t.Fatalf("%s.serviceTier Kind = %s, want %s", typeName, fields["serviceTier"].Kind, FieldPlanNullableServiceTier)
		}
	}
	thread := selectedByName["Thread"]
	threadFields := map[string]FieldPlan{}
	for _, field := range thread.Fields {
		threadFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"agentNickname": "*protocolv2.Nullable[string]",
		"agentRole":     "*protocolv2.Nullable[string]",
		"cliVersion":    "string",
		"createdAt":     "int64",
		"cwd":           "string",
		"ephemeral":     "bool",
		"forkedFromId":  "*protocolv2.Nullable[string]",
		"gitInfo":       "*protocolv2.Nullable[GitInfo]",
		"id":            "string",
		"modelProvider": "string",
		"name":          "*protocolv2.Nullable[string]",
		"path":          "*protocolv2.Nullable[string]",
		"preview":       "string",
		"sessionId":     "string",
		"source":        "SessionSource",
		"status":        "ThreadStatus",
		"threadSource":  "*protocolv2.Nullable[ThreadSource]",
		"turns":         "[]Turn",
		"updatedAt":     "int64",
	} {
		if got := threadFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("Thread.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	for _, fieldName := range []string{"cliVersion", "createdAt", "cwd", "ephemeral", "id", "modelProvider", "preview", "sessionId", "source", "status", "turns", "updatedAt"} {
		if !threadFields[fieldName].Required {
			t.Fatalf("Thread.%s must remain required", fieldName)
		}
	}
	activeProfile := selectedByName["ActivePermissionProfile"]
	activeProfileFields := map[string]FieldPlan{}
	for _, field := range activeProfile.Fields {
		activeProfileFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"extends": "*protocolv2.Nullable[string]",
		"id":      "string",
	} {
		if got := activeProfileFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("ActivePermissionProfile.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	if !activeProfileFields["id"].Required {
		t.Fatal("ActivePermissionProfile.id must remain required")
	}
	gitInfo := selectedByName["GitInfo"]
	gitInfoFields := map[string]FieldPlan{}
	for _, field := range gitInfo.Fields {
		gitInfoFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"branch":    "*protocolv2.Nullable[string]",
		"originUrl": "*protocolv2.Nullable[string]",
		"sha":       "*protocolv2.Nullable[string]",
	} {
		if got := gitInfoFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("GitInfo.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	threadListResponse := selectedByName["ThreadListResponse"]
	threadListFields := map[string]FieldPlan{}
	for _, field := range threadListResponse.Fields {
		threadListFields[field.FieldName] = field
	}
	if threadListFields["data"].GoType != "[]Thread" || threadListFields["nextCursor"].GoType != "*protocolv2.Nullable[string]" {
		t.Fatalf("ThreadListResponse fields = %#v", threadListFields)
	}
	for _, typeName := range []string{"ThreadReadResponse", "ThreadRollbackResponse", "ThreadMetadataUpdateResponse", "ThreadUnarchiveResponse", "ThreadStartedNotification"} {
		payload := selectedByName[typeName]
		fields := map[string]FieldPlan{}
		for _, field := range payload.Fields {
			fields[field.FieldName] = field
		}
		if fields["thread"].GoType != "Thread" || !fields["thread"].Required {
			t.Fatalf("%s.thread = %#v", typeName, fields["thread"])
		}
	}
	threadStatusChanged := selectedByName["ThreadStatusChangedNotification"]
	threadStatusChangedFields := map[string]FieldPlan{}
	for _, field := range threadStatusChanged.Fields {
		threadStatusChangedFields[field.FieldName] = field
	}
	if threadStatusChangedFields["status"].GoType != "ThreadStatus" || !threadStatusChangedFields["status"].Required ||
		threadStatusChangedFields["threadId"].GoType != "string" || !threadStatusChangedFields["threadId"].Required {
		t.Fatalf("ThreadStatusChangedNotification fields = %#v", threadStatusChangedFields)
	}
	tokenUsageBreakdownFields := generatedFields("TokenUsageBreakdown")
	for _, fieldName := range []string{"cachedInputTokens", "inputTokens", "outputTokens", "reasoningOutputTokens", "totalTokens"} {
		if tokenUsageBreakdownFields[fieldName].GoType != "int64" || !tokenUsageBreakdownFields[fieldName].Required {
			t.Fatalf("TokenUsageBreakdown.%s field = %#v", fieldName, tokenUsageBreakdownFields[fieldName])
		}
	}
	threadTokenUsageFields := generatedFields("ThreadTokenUsage")
	if threadTokenUsageFields["last"].GoType != "TokenUsageBreakdown" || !threadTokenUsageFields["last"].Required ||
		threadTokenUsageFields["total"].GoType != "TokenUsageBreakdown" || !threadTokenUsageFields["total"].Required ||
		threadTokenUsageFields["modelContextWindow"].GoType != "*protocolv2.Nullable[int64]" {
		t.Fatalf("ThreadTokenUsage fields = %#v", threadTokenUsageFields)
	}
	threadTokenUsageUpdatedFields := generatedFields("ThreadTokenUsageUpdatedNotification")
	if threadTokenUsageUpdatedFields["threadId"].GoType != "string" || !threadTokenUsageUpdatedFields["threadId"].Required ||
		threadTokenUsageUpdatedFields["turnId"].GoType != "string" || !threadTokenUsageUpdatedFields["turnId"].Required ||
		threadTokenUsageUpdatedFields["tokenUsage"].GoType != "ThreadTokenUsage" || !threadTokenUsageUpdatedFields["tokenUsage"].Required {
		t.Fatalf("ThreadTokenUsageUpdatedNotification fields = %#v", threadTokenUsageUpdatedFields)
	}
	turnPlanStepFields := generatedFields("TurnPlanStep")
	if turnPlanStepFields["status"].GoType != "TurnPlanStepStatus" || !turnPlanStepFields["status"].Required ||
		turnPlanStepFields["step"].GoType != "string" || !turnPlanStepFields["step"].Required {
		t.Fatalf("TurnPlanStep fields = %#v", turnPlanStepFields)
	}
	turnPlanUpdatedFields := generatedFields("TurnPlanUpdatedNotification")
	if turnPlanUpdatedFields["explanation"].GoType != "*protocolv2.Nullable[string]" ||
		turnPlanUpdatedFields["plan"].GoType != "[]TurnPlanStep" || !turnPlanUpdatedFields["plan"].Required ||
		turnPlanUpdatedFields["threadId"].GoType != "string" || !turnPlanUpdatedFields["threadId"].Required ||
		turnPlanUpdatedFields["turnId"].GoType != "string" || !turnPlanUpdatedFields["turnId"].Required {
		t.Fatalf("TurnPlanUpdatedNotification fields = %#v", turnPlanUpdatedFields)
	}
	turnStartResponse := selectedByName["TurnStartResponse"]
	turnStartResponseFields := map[string]FieldPlan{}
	for _, field := range turnStartResponse.Fields {
		turnStartResponseFields[field.FieldName] = field
	}
	if turnStartResponseFields["turn"].GoType != "Turn" || !turnStartResponseFields["turn"].Required {
		t.Fatalf("TurnStartResponse.turn = %#v", turnStartResponseFields["turn"])
	}
	turn := selectedByName["Turn"]
	turnFields := map[string]FieldPlan{}
	for _, field := range turn.Fields {
		turnFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"error":  "*protocolv2.Nullable[TurnError]",
		"id":     "string",
		"items":  "[]ThreadItem",
		"status": "TurnStatus",
	} {
		if got := turnFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("Turn.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	if !turnFields["id"].Required || !turnFields["items"].Required || !turnFields["status"].Required {
		t.Fatalf("Turn required fields = id:%t items:%t status:%t", turnFields["id"].Required, turnFields["items"].Required, turnFields["status"].Required)
	}
	mcpToolCallResult := selectedByName["McpToolCallResult"]
	mcpToolCallResultFields := map[string]FieldPlan{}
	for _, field := range mcpToolCallResult.Fields {
		mcpToolCallResultFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"_meta":             "*protocolv2.JSONValue",
		"content":           "[]protocolv2.JSONValue",
		"structuredContent": "*protocolv2.JSONValue",
	} {
		if got := mcpToolCallResultFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("McpToolCallResult.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	if mcpToolCallResultFields["_meta"].Kind != FieldPlanJSONValue ||
		mcpToolCallResultFields["content"].Kind != FieldPlanArrayJSONValue ||
		mcpToolCallResultFields["structuredContent"].Kind != FieldPlanJSONValue {
		t.Fatalf("McpToolCallResult JSONValue fields = %#v", mcpToolCallResultFields)
	}
	turnSteer := selectedByName["TurnSteerParams"]
	turnSteerFields := map[string]FieldPlan{}
	for _, field := range turnSteer.Fields {
		turnSteerFields[field.FieldName] = field
	}
	if turnSteerFields["input"].GoType != "[]UserInput" || !turnSteerFields["input"].Required {
		t.Fatalf("TurnSteerParams.input = %#v", turnSteerFields["input"])
	}
	if turnSteerFields["responsesapiClientMetadata"].GoType != "*protocolv2.Nullable[map[string]string]" {
		t.Fatalf("TurnSteerParams.responsesapiClientMetadata GoType = %q, want Nullable map", turnSteerFields["responsesapiClientMetadata"].GoType)
	}
	textElement := selectedByName["TextElement"]
	textElementFields := map[string]FieldPlan{}
	for _, field := range textElement.Fields {
		textElementFields[field.FieldName] = field
	}
	if textElementFields["byteRange"].GoType != "ByteRange" || !textElementFields["byteRange"].Required {
		t.Fatalf("TextElement.byteRange = %#v", textElementFields["byteRange"])
	}
	if textElementFields["placeholder"].GoType != "*protocolv2.Nullable[string]" {
		t.Fatalf("TextElement.placeholder GoType = %q, want Nullable string", textElementFields["placeholder"].GoType)
	}
	for _, name := range []string{"ClientRequest", "JSONRPCMessage", "ServerNotification", "ServerRequest"} {
		if _, ok := selectedByName[name]; ok {
			t.Fatalf("aggregate type %s should not be first-pass generated", name)
		}
	}
}

func TestJSONRPCMessageIsNotPublicGeneratedSurface(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	selected, err := SelectFirstPassGeneratedTypes(plan)
	if err != nil {
		t.Fatal(err)
	}
	for _, typ := range selected {
		if strings.HasPrefix(typ.TypeName, "JSONRPC") {
			t.Fatalf("JSON-RPC envelope type %s must not be public generated protocolv2 surface", typ.TypeName)
		}
	}
}

func TestAppInfoDefinitionsStayEquivalent(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	response, ok := plan.TypeBySchema("v2/AppsListResponse.json")
	if !ok {
		t.Fatal("AppsListResponse.json schema was not loaded")
	}
	notification, ok := plan.TypeBySchema("v2/AppListUpdatedNotification.json")
	if !ok {
		t.Fatal("AppListUpdatedNotification.json schema was not loaded")
	}
	for _, name := range []string{"AppBranding", "AppInfo", "AppMetadata", "AppReview", "AppScreenshot"} {
		responseDefinition := response.Schema.Definitions[name]
		notificationDefinition := notification.Schema.Definitions[name]
		if responseDefinition == nil || notificationDefinition == nil {
			t.Fatalf("%s definition is missing", name)
		}
		responseJSON, err := json.Marshal(responseDefinition)
		if err != nil {
			t.Fatal(err)
		}
		notificationJSON, err := json.Marshal(notificationDefinition)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(responseJSON, notificationJSON) {
			t.Fatalf("%s definitions diverged between app list response and notification", name)
		}
	}
}

func TestFuzzyFileSearchResultDefinitionsStayEquivalent(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	response, ok := plan.TypeBySchema("FuzzyFileSearchResponse.json")
	if !ok {
		t.Fatal("FuzzyFileSearchResponse.json schema was not loaded")
	}
	updated, ok := plan.TypeBySchema("FuzzyFileSearchSessionUpdatedNotification.json")
	if !ok {
		t.Fatal("FuzzyFileSearchSessionUpdatedNotification.json schema was not loaded")
	}
	responseDefinition := response.Schema.Definitions["FuzzyFileSearchResult"]
	updatedDefinition := updated.Schema.Definitions["FuzzyFileSearchResult"]
	if responseDefinition == nil || updatedDefinition == nil {
		t.Fatal("FuzzyFileSearchResult definition is missing")
	}
	responseJSON, err := json.Marshal(responseDefinition)
	if err != nil {
		t.Fatal(err)
	}
	updatedJSON, err := json.Marshal(updatedDefinition)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(responseJSON, updatedJSON) {
		t.Fatal("FuzzyFileSearchResult definitions diverged between response and session update notification")
	}
	responseEnum := response.Schema.Definitions["FuzzyFileSearchMatchType"]
	updatedEnum := updated.Schema.Definitions["FuzzyFileSearchMatchType"]
	if responseEnum == nil || updatedEnum == nil {
		t.Fatal("FuzzyFileSearchMatchType definition is missing")
	}
	responseEnumJSON, err := json.Marshal(responseEnum)
	if err != nil {
		t.Fatal(err)
	}
	updatedEnumJSON, err := json.Marshal(updatedEnum)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(responseEnumJSON, updatedEnumJSON) {
		t.Fatal("FuzzyFileSearchMatchType definitions diverged between response and session update notification")
	}
}

func TestProcessTerminalSizeDefinitionsStayEquivalent(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	spawn, ok := plan.TypeBySchema("v2/ProcessSpawnParams.json")
	if !ok {
		t.Fatal("ProcessSpawnParams.json schema was not loaded")
	}
	resize, ok := plan.TypeBySchema("v2/ProcessResizePtyParams.json")
	if !ok {
		t.Fatal("ProcessResizePtyParams.json schema was not loaded")
	}
	spawnDefinition := spawn.Schema.Definitions["ProcessTerminalSize"]
	resizeDefinition := resize.Schema.Definitions["ProcessTerminalSize"]
	if spawnDefinition == nil || resizeDefinition == nil {
		t.Fatal("ProcessTerminalSize definition is missing")
	}
	spawnJSON, err := json.Marshal(spawnDefinition)
	if err != nil {
		t.Fatal(err)
	}
	resizeJSON, err := json.Marshal(resizeDefinition)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(spawnJSON, resizeJSON) {
		t.Fatal("ProcessTerminalSize definitions diverged between process spawn and resize params")
	}
}

func TestCommandExecTerminalSizeDefinitionsStayEquivalent(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	exec, ok := plan.TypeBySchema("v2/CommandExecParams.json")
	if !ok {
		t.Fatal("CommandExecParams.json schema was not loaded")
	}
	resize, ok := plan.TypeBySchema("v2/CommandExecResizeParams.json")
	if !ok {
		t.Fatal("CommandExecResizeParams.json schema was not loaded")
	}
	if !bytes.Equal(encodedDefinition(t, exec, "CommandExecTerminalSize"), encodedDefinition(t, resize, "CommandExecTerminalSize")) {
		t.Fatal("CommandExecTerminalSize definitions diverged between command exec and resize params")
	}
}

func TestExternalAgentConfigMigrationDefinitionsStayEquivalent(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	detect, ok := plan.TypeBySchema("v2/ExternalAgentConfigDetectResponse.json")
	if !ok {
		t.Fatal("ExternalAgentConfigDetectResponse.json schema was not loaded")
	}
	importParams, ok := plan.TypeBySchema("v2/ExternalAgentConfigImportParams.json")
	if !ok {
		t.Fatal("ExternalAgentConfigImportParams.json schema was not loaded")
	}
	for _, name := range []string{
		"CommandMigration",
		"ExternalAgentConfigMigrationItem",
		"ExternalAgentConfigMigrationItemType",
		"HookMigration",
		"McpServerMigration",
		"MigrationDetails",
		"PluginsMigration",
		"SessionMigration",
		"SubagentMigration",
	} {
		if !bytes.Equal(encodedDefinition(t, detect, name), encodedDefinition(t, importParams, name)) {
			t.Fatalf("%s definitions diverged between external agent config detect and import", name)
		}
	}
}

func TestSelectGeneratedEnums(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	enums, err := SelectGeneratedEnums(plan)
	if err != nil {
		t.Fatal(err)
	}
	if got, min := len(enums), 105; got < min {
		t.Fatalf("selected generated enum count = %d, want at least %d", got, min)
	}
	enumByName := map[string]EnumPlan{}
	for _, enum := range enums {
		enumByName[enum.TypeName] = enum
	}
	for _, name := range []string{
		"AddCreditsNudgeCreditType",
		"AuthMode",
		"ApprovalsReviewer",
		"CancelLoginAccountStatus",
		"ChatgptAuthTokensRefreshReason",
		"CommandExecOutputStream",
		"ConsumeAccountRateLimitResetCreditOutcome",
		"ConversationTextRole",
		"ExperimentalFeatureStage",
		"ExternalAgentConfigMigrationItemType",
		"FileChangeApprovalDecision",
		"ImageDetail",
		"InputModality",
		"LocalShellStatus",
		"McpAuthStatus",
		"ModelRerouteReason",
		"ModelVerification",
		"ModeKind",
		"NetworkDomainPermission",
		"NetworkUnixSocketPermission",
		"NetworkAccess",
		"Personality",
		"PluginAvailability",
		"ProcessOutputStream",
		"ReasoningSummary",
		"ResidencyRequirement",
		"ThreadMemoryMode",
		"ThreadStartSource",
		"ThreadUnsubscribeStatus",
		"WindowsSandboxReadiness",
	} {
		if _, ok := enumByName[name]; !ok {
			t.Fatalf("expected generated enum %s", name)
		}
	}
	cancelStatus := enumByName["CancelLoginAccountStatus"]
	if got, want := strings.Join(cancelStatus.Values, ","), "canceled,notFound"; got != want {
		t.Fatalf("CancelLoginAccountStatus values = %s, want %s", got, want)
	}
	refreshReason := enumByName["ChatgptAuthTokensRefreshReason"]
	if got, want := strings.Join(refreshReason.Values, ","), "unauthorized"; got != want {
		t.Fatalf("ChatgptAuthTokensRefreshReason values = %s, want %s", got, want)
	}
	fileChangeDecision := enumByName["FileChangeApprovalDecision"]
	if got, want := strings.Join(fileChangeDecision.Values, ","), "accept,acceptForSession,decline,cancel"; got != want {
		t.Fatalf("FileChangeApprovalDecision values = %s, want %s", got, want)
	}
	experimentalFeatureStage := enumByName["ExperimentalFeatureStage"]
	if got, want := strings.Join(experimentalFeatureStage.Values, ","), "beta,underDevelopment,stable,deprecated,removed"; got != want {
		t.Fatalf("ExperimentalFeatureStage values = %s, want %s", got, want)
	}
	networkDomainPermission := enumByName["NetworkDomainPermission"]
	if got, want := strings.Join(networkDomainPermission.Values, ","), "allow,deny"; got != want {
		t.Fatalf("NetworkDomainPermission values = %s, want %s", got, want)
	}
	networkUnixSocketPermission := enumByName["NetworkUnixSocketPermission"]
	if got, want := strings.Join(networkUnixSocketPermission.Values, ","), "allow,deny"; got != want {
		t.Fatalf("NetworkUnixSocketPermission values = %s, want %s", got, want)
	}
	residencyRequirement := enumByName["ResidencyRequirement"]
	if got, want := strings.Join(residencyRequirement.Values, ","), "us"; got != want {
		t.Fatalf("ResidencyRequirement values = %s, want %s", got, want)
	}
	reasoningSummary := enumByName["ReasoningSummary"]
	if got, want := strings.Join(reasoningSummary.Values, ","), "auto,concise,detailed,none"; got != want {
		t.Fatalf("ReasoningSummary values = %s, want %s", got, want)
	}
	modeKind := enumByName["ModeKind"]
	if got, want := strings.Join(modeKind.Values, ","), "plan,default"; got != want {
		t.Fatalf("ModeKind values = %s, want %s", got, want)
	}
	personality := enumByName["Personality"]
	if got, want := strings.Join(personality.Values, ","), "none,friendly,pragmatic"; got != want {
		t.Fatalf("Personality values = %s, want %s", got, want)
	}
	imageDetail := enumByName["ImageDetail"]
	if got, want := strings.Join(imageDetail.Values, ","), "auto,low,high,original"; got != want {
		t.Fatalf("ImageDetail values = %s, want %s", got, want)
	}
	localShellStatus := enumByName["LocalShellStatus"]
	if got, want := strings.Join(localShellStatus.Values, ","), "completed,in_progress,incomplete"; got != want {
		t.Fatalf("LocalShellStatus values = %s, want %s", got, want)
	}
	inputModality := enumByName["InputModality"]
	if got, want := strings.Join(inputModality.Values, ","), "text,image"; got != want {
		t.Fatalf("InputModality values = %s, want %s", got, want)
	}
	processOutputStream := enumByName["ProcessOutputStream"]
	if got, want := strings.Join(processOutputStream.Values, ","), "stdout,stderr"; got != want {
		t.Fatalf("ProcessOutputStream values = %s, want %s", got, want)
	}
	pluginAvailability := enumByName["PluginAvailability"]
	if got, want := strings.Join(pluginAvailability.Values, ","), "DISABLED_BY_ADMIN,AVAILABLE"; got != want {
		t.Fatalf("PluginAvailability values = %s, want %s", got, want)
	}
	commandExecOutputStream := enumByName["CommandExecOutputStream"]
	if got, want := strings.Join(commandExecOutputStream.Values, ","), "stdout,stderr"; got != want {
		t.Fatalf("CommandExecOutputStream values = %s, want %s", got, want)
	}
}

func TestStringEnumValuesRejectsImpureSingleOneOfWrapper(t *testing.T) {
	stringEnum := func(value string) *Schema {
		return &Schema{
			Type: SchemaTypeSet{Values: []string{"string"}},
			Enum: []string{value},
		}
	}
	if values, ok := stringEnumValues(&Schema{OneOf: []*Schema{stringEnum("known")}}); !ok || strings.Join(values, ",") != "known" {
		t.Fatalf("pure single-oneOf enum values = %v, ok=%t", values, ok)
	}

	trueAdditionalProperties := true
	defaultWrapped := mustParseSchema(t, `{"oneOf":[{"type":"string","enum":["known"]}],"default":"known"}`)
	requiredWrapped := mustParseSchema(t, `{"oneOf":[{"type":"string","enum":["known"]}],"required":["value"]}`)
	cases := map[string]*Schema{
		"outer anyOf": {
			OneOf: []*Schema{stringEnum("known")},
			AnyOf: []*Schema{stringEnum("other")},
		},
		"outer ref": {
			OneOf: []*Schema{stringEnum("known")},
			Ref:   "#/definitions/Other",
		},
		"outer properties": {
			OneOf:      []*Schema{stringEnum("known")},
			Properties: map[string]*Schema{"extra": stringEnum("other")},
		},
		"outer additionalProperties": {
			OneOf: []*Schema{stringEnum("known")},
			AdditionalProperties: AdditionalProperties{
				Bool:    &trueAdditionalProperties,
				Present: true,
			},
		},
		"outer type": {
			OneOf: []*Schema{stringEnum("known")},
			Type:  SchemaTypeSet{Values: []string{"string"}},
		},
		"outer default":  defaultWrapped,
		"outer required": requiredWrapped,
	}
	for name, schema := range cases {
		t.Run(name, func(t *testing.T) {
			if values, ok := stringEnumValues(schema); ok {
				t.Fatalf("impure wrapper was accepted as enum: %v", values)
			}
		})
	}
}

func TestReviewedStringEnumValuesAcceptsCheckpointedPureMultiOneOf(t *testing.T) {
	stringEnum := func(value string) *Schema {
		return &Schema{
			Type: SchemaTypeSet{Values: []string{"string"}},
			Enum: []string{value},
		}
	}
	schema := &Schema{OneOf: []*Schema{
		stringEnum("accept"),
		stringEnum("decline"),
	}}
	if values, ok := reviewedStringEnumValues("FileChangeRequestApprovalResponse.json", "FileChangeApprovalDecision", schema); !ok || strings.Join(values, ",") != "accept,decline" {
		t.Fatalf("reviewed multi-oneOf enum values = %v, ok=%t", values, ok)
	}
	modelSchema := &Schema{OneOf: []*Schema{
		stringEnum("text"),
		stringEnum("image"),
	}}
	if values, ok := reviewedStringEnumValues("v2/ModelListResponse.json", "InputModality", modelSchema); !ok || strings.Join(values, ",") != "text,image" {
		t.Fatalf("reviewed InputModality multi-oneOf enum values = %v, ok=%t", values, ok)
	}
	processSchema := &Schema{OneOf: []*Schema{
		stringEnum("stdout"),
		stringEnum("stderr"),
	}}
	if values, ok := reviewedStringEnumValues("v2/ProcessOutputDeltaNotification.json", "ProcessOutputStream", processSchema); !ok || strings.Join(values, ",") != "stdout,stderr" {
		t.Fatalf("reviewed ProcessOutputStream multi-oneOf enum values = %v, ok=%t", values, ok)
	}
	if values, ok := reviewedStringEnumValues("v2/CommandExecOutputDeltaNotification.json", "CommandExecOutputStream", processSchema); !ok || strings.Join(values, ",") != "stdout,stderr" {
		t.Fatalf("reviewed CommandExecOutputStream multi-oneOf enum values = %v, ok=%t", values, ok)
	}
	if values, ok := reviewedStringEnumValues("Other.json", "FileChangeApprovalDecision", schema); ok {
		t.Fatalf("uncheckpointed multi-oneOf enum was accepted: %v", values)
	}
	mixed := &Schema{OneOf: []*Schema{
		stringEnum("accept"),
		{
			Type:       SchemaTypeSet{Values: []string{"object"}},
			Properties: map[string]*Schema{"value": stringEnum("decline")},
		},
	}}
	if values, ok := reviewedStringEnumValues("FileChangeRequestApprovalResponse.json", "FileChangeApprovalDecision", mixed); ok {
		t.Fatalf("mixed multi-oneOf enum was accepted: %v", values)
	}
}

func mustParseSchema(t *testing.T, raw string) *Schema {
	t.Helper()
	var schema Schema
	if err := json.Unmarshal([]byte(raw), &schema); err != nil {
		t.Fatal(err)
	}
	return &schema
}

func TestSelectGeneratedScalarUnions(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	unions, err := SelectGeneratedScalarUnions(plan)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(unions), 4; got != want {
		t.Fatalf("selected generated scalar union count = %d, want %d", got, want)
	}
	unionByName := map[string]ScalarUnionPlan{}
	for _, union := range unions {
		unionByName[union.TypeName] = union
	}
	requestID := unionByName["RequestId"]
	if requestID.TypeName != "RequestId" {
		t.Fatalf("scalar union type = %q, want RequestId", requestID.TypeName)
	}
	if got, want := len(requestID.Variants), 2; got != want {
		t.Fatalf("RequestId variant count = %d, want %d", got, want)
	}
	seen := map[string]string{}
	for _, variant := range requestID.Variants {
		seen[variant.JSONKind] = variant.GoType
	}
	if seen["string"] != "string" || seen["number"] != "int64" {
		t.Fatalf("RequestId variants = %#v", requestID.Variants)
	}
	functionOutput := unionByName["FunctionCallOutputBody"]
	if functionOutput.TypeName != "FunctionCallOutputBody" {
		t.Fatalf("scalar union type = %q, want FunctionCallOutputBody", functionOutput.TypeName)
	}
	if got, want := len(functionOutput.Variants), 2; got != want {
		t.Fatalf("FunctionCallOutputBody variant count = %d, want %d", got, want)
	}
	functionOutputVariants := map[string]string{}
	for _, variant := range functionOutput.Variants {
		functionOutputVariants[variant.JSONKind] = variant.GoType
	}
	if functionOutputVariants["string"] != "string" || functionOutputVariants["array"] != "[]FunctionCallOutputContentItem" {
		t.Fatalf("FunctionCallOutputBody variants = %#v", functionOutput.Variants)
	}
	threadListCwdFilter := unionByName["ThreadListCwdFilter"]
	if threadListCwdFilter.TypeName != "ThreadListCwdFilter" {
		t.Fatalf("scalar union type = %q, want ThreadListCwdFilter", threadListCwdFilter.TypeName)
	}
	if got, want := len(threadListCwdFilter.Variants), 2; got != want {
		t.Fatalf("ThreadListCwdFilter variant count = %d, want %d", got, want)
	}
	threadListCwdVariants := map[string]string{}
	for _, variant := range threadListCwdFilter.Variants {
		threadListCwdVariants[variant.JSONKind] = variant.GoType
	}
	if threadListCwdVariants["string"] != "string" || threadListCwdVariants["array"] != "[]string" {
		t.Fatalf("ThreadListCwdFilter variants = %#v", threadListCwdFilter.Variants)
	}
}

func TestSelectGeneratedMixedUnions(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	unions, err := SelectGeneratedMixedUnions(plan)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(unions), 8; got != want {
		t.Fatalf("selected generated mixed union count = %d, want %d", got, want)
	}
	unionByName := map[string]MixedUnionPlan{}
	for _, union := range unions {
		unionByName[union.TypeName] = union
	}
	askForApproval := unionByName["AskForApproval"]
	if askForApproval.TypeName != "AskForApproval" {
		t.Fatalf("mixed union type = %q, want AskForApproval", askForApproval.TypeName)
	}
	if got, want := len(askForApproval.Variants), 5; got != want {
		t.Fatalf("AskForApproval variant count = %d, want %d", got, want)
	}
	var granularApproval MixedUnionVariantPlan
	for _, variant := range askForApproval.Variants {
		switch variant.DiscriminatorValue {
		case "untrusted", "on-failure", "on-request", "never":
			if variant.JSONKind != "string" || len(variant.Fields) != 0 {
				t.Fatalf("AskForApproval string variant = %#v", variant)
			}
		case "granular":
			granularApproval = variant
		default:
			t.Fatalf("unexpected AskForApproval variant = %#v", variant)
		}
	}
	if granularApproval.PayloadTypeName != "AskForApprovalGranular" || granularApproval.JSONKind != "object" {
		t.Fatalf("AskForApproval granular variant = %#v", granularApproval)
	}
	granularApprovalFields := map[string]FieldPlan{}
	for _, field := range granularApproval.Fields {
		granularApprovalFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"mcp_elicitations":    "bool",
		"request_permissions": "*bool",
		"rules":               "bool",
		"sandbox_approval":    "bool",
		"skill_approval":      "*bool",
	} {
		if got := granularApprovalFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("AskForApproval.granular.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}

	messagePhase := unionByName["MessagePhase"]
	if messagePhase.TypeName != "MessagePhase" {
		t.Fatalf("mixed union type = %q, want MessagePhase", messagePhase.TypeName)
	}
	if got, want := len(messagePhase.Variants), 2; got != want {
		t.Fatalf("MessagePhase variant count = %d, want %d", got, want)
	}
	for _, variant := range messagePhase.Variants {
		if variant.JSONKind != "string" || len(variant.Fields) != 0 {
			t.Fatalf("MessagePhase variant = %#v", variant)
		}
	}

	review := unionByName["ReviewDecision"]
	if review.TypeName != "ReviewDecision" {
		t.Fatalf("mixed union type = %q, want ReviewDecision", review.TypeName)
	}
	if got, want := len(review.Variants), 7; got != want {
		t.Fatalf("ReviewDecision variant count = %d, want %d", got, want)
	}
	var execPolicyVariant MixedUnionVariantPlan
	var networkPolicyVariant MixedUnionVariantPlan
	for _, variant := range review.Variants {
		switch variant.DiscriminatorValue {
		case "approved_execpolicy_amendment":
			execPolicyVariant = variant
		case "network_policy_amendment":
			networkPolicyVariant = variant
		}
	}
	if execPolicyVariant.PayloadTypeName != "ReviewDecisionApprovedExecpolicyAmendment" {
		t.Fatalf("exec policy payload = %q", execPolicyVariant.PayloadTypeName)
	}
	if got, want := len(execPolicyVariant.Fields), 1; got != want {
		t.Fatalf("exec policy field count = %d, want %d", got, want)
	}
	if execPolicyVariant.Fields[0].FieldName != "proposed_execpolicy_amendment" || execPolicyVariant.Fields[0].GoType != "[]string" {
		t.Fatalf("exec policy field = %#v", execPolicyVariant.Fields[0])
	}
	if networkPolicyVariant.PayloadTypeName != "ReviewDecisionNetworkPolicyAmendment" {
		t.Fatalf("network policy payload = %q", networkPolicyVariant.PayloadTypeName)
	}
	if got, want := len(networkPolicyVariant.Fields), 1; got != want {
		t.Fatalf("network policy field count = %d, want %d", got, want)
	}
	if networkPolicyVariant.Fields[0].FieldName != "network_policy_amendment" || networkPolicyVariant.Fields[0].GoType != "NetworkPolicyAmendment" {
		t.Fatalf("network policy field = %#v", networkPolicyVariant.Fields[0])
	}

	commandDecision := unionByName["CommandExecutionApprovalDecision"]
	if commandDecision.TypeName != "CommandExecutionApprovalDecision" {
		t.Fatalf("mixed union type = %q, want CommandExecutionApprovalDecision", commandDecision.TypeName)
	}
	if got, want := len(commandDecision.Variants), 6; got != want {
		t.Fatalf("CommandExecutionApprovalDecision variant count = %d, want %d", got, want)
	}
	var acceptWithExecPolicy MixedUnionVariantPlan
	var applyNetworkPolicy MixedUnionVariantPlan
	for _, variant := range commandDecision.Variants {
		switch variant.DiscriminatorValue {
		case "acceptWithExecpolicyAmendment":
			acceptWithExecPolicy = variant
		case "applyNetworkPolicyAmendment":
			applyNetworkPolicy = variant
		}
	}
	if acceptWithExecPolicy.PayloadTypeName != "CommandExecutionApprovalDecisionAcceptWithExecpolicyAmendment" {
		t.Fatalf("acceptWithExecpolicyAmendment payload = %q", acceptWithExecPolicy.PayloadTypeName)
	}
	if got, want := len(acceptWithExecPolicy.Fields), 1; got != want {
		t.Fatalf("acceptWithExecpolicyAmendment field count = %d, want %d", got, want)
	}
	if acceptWithExecPolicy.Fields[0].FieldName != "execpolicy_amendment" || acceptWithExecPolicy.Fields[0].GoType != "[]string" {
		t.Fatalf("acceptWithExecpolicyAmendment field = %#v", acceptWithExecPolicy.Fields[0])
	}
	if applyNetworkPolicy.PayloadTypeName != "CommandExecutionApprovalDecisionApplyNetworkPolicyAmendment" {
		t.Fatalf("applyNetworkPolicyAmendment payload = %q", applyNetworkPolicy.PayloadTypeName)
	}
	if got, want := len(applyNetworkPolicy.Fields), 1; got != want {
		t.Fatalf("applyNetworkPolicyAmendment field count = %d, want %d", got, want)
	}
	if applyNetworkPolicy.Fields[0].FieldName != "network_policy_amendment" || applyNetworkPolicy.Fields[0].GoType != "NetworkPolicyAmendment" {
		t.Fatalf("applyNetworkPolicyAmendment field = %#v", applyNetworkPolicy.Fields[0])
	}

	codexError := unionByName["CodexErrorInfo"]
	if codexError.TypeName != "CodexErrorInfo" {
		t.Fatalf("mixed union type = %q, want CodexErrorInfo", codexError.TypeName)
	}
	if got, want := len(codexError.Variants), 15; got != want {
		t.Fatalf("CodexErrorInfo variant count = %d, want %d", got, want)
	}
	var activeTurnNotSteerable MixedUnionVariantPlan
	for _, variant := range codexError.Variants {
		if variant.DiscriminatorValue == "activeTurnNotSteerable" {
			activeTurnNotSteerable = variant
		}
	}
	if activeTurnNotSteerable.PayloadTypeName != "CodexErrorInfoActiveTurnNotSteerable" {
		t.Fatalf("activeTurnNotSteerable payload = %q", activeTurnNotSteerable.PayloadTypeName)
	}
	if got, want := len(activeTurnNotSteerable.Fields), 1; got != want {
		t.Fatalf("activeTurnNotSteerable field count = %d, want %d", got, want)
	}
	if activeTurnNotSteerable.Fields[0].FieldName != "turnKind" || activeTurnNotSteerable.Fields[0].GoType != "NonSteerableTurnKind" {
		t.Fatalf("activeTurnNotSteerable field = %#v", activeTurnNotSteerable.Fields[0])
	}

	turnItemsView := unionByName["TurnItemsView"]
	if turnItemsView.TypeName != "TurnItemsView" {
		t.Fatalf("mixed union type = %q, want TurnItemsView", turnItemsView.TypeName)
	}
	if got, want := len(turnItemsView.Variants), 3; got != want {
		t.Fatalf("TurnItemsView variant count = %d, want %d", got, want)
	}
	for _, variant := range turnItemsView.Variants {
		if variant.JSONKind != "string" || len(variant.Fields) != 0 {
			t.Fatalf("TurnItemsView variant = %#v", variant)
		}
	}

	sessionSource := unionByName["SessionSource"]
	if sessionSource.TypeName != "SessionSource" {
		t.Fatalf("mixed union type = %q, want SessionSource", sessionSource.TypeName)
	}
	if got, want := len(sessionSource.Variants), 7; got != want {
		t.Fatalf("SessionSource variant count = %d, want %d", got, want)
	}
	var customSessionSource MixedUnionVariantPlan
	var subAgentSessionSource MixedUnionVariantPlan
	for _, variant := range sessionSource.Variants {
		switch variant.DiscriminatorValue {
		case "cli", "vscode", "exec", "appServer", "unknown":
			if variant.JSONKind != "string" || len(variant.Fields) != 0 || variant.DirectValueField != nil {
				t.Fatalf("SessionSource string variant = %#v", variant)
			}
		case "custom":
			customSessionSource = variant
		case "subAgent":
			subAgentSessionSource = variant
		default:
			t.Fatalf("unexpected SessionSource variant = %#v", variant)
		}
	}
	if customSessionSource.DirectValueField == nil ||
		customSessionSource.DirectValueField.FieldName != "custom" ||
		customSessionSource.DirectValueField.GoType != "string" {
		t.Fatalf("SessionSource custom direct field = %#v", customSessionSource.DirectValueField)
	}
	if subAgentSessionSource.DirectValueField == nil ||
		subAgentSessionSource.DirectValueField.FieldName != "subAgent" ||
		subAgentSessionSource.DirectValueField.GoType != "SubAgentSource" {
		t.Fatalf("SessionSource subAgent direct field = %#v", subAgentSessionSource.DirectValueField)
	}

	subAgentSource := unionByName["SubAgentSource"]
	if subAgentSource.TypeName != "SubAgentSource" {
		t.Fatalf("mixed union type = %q, want SubAgentSource", subAgentSource.TypeName)
	}
	if got, want := len(subAgentSource.Variants), 5; got != want {
		t.Fatalf("SubAgentSource variant count = %d, want %d", got, want)
	}
	var threadSpawnSubAgent MixedUnionVariantPlan
	var otherSubAgent MixedUnionVariantPlan
	for _, variant := range subAgentSource.Variants {
		switch variant.DiscriminatorValue {
		case "review", "compact", "memory_consolidation":
			if variant.JSONKind != "string" || len(variant.Fields) != 0 || variant.DirectValueField != nil {
				t.Fatalf("SubAgentSource string variant = %#v", variant)
			}
		case "thread_spawn":
			threadSpawnSubAgent = variant
		case "other":
			otherSubAgent = variant
		default:
			t.Fatalf("unexpected SubAgentSource variant = %#v", variant)
		}
	}
	threadSpawnFields := map[string]FieldPlan{}
	for _, field := range threadSpawnSubAgent.Fields {
		threadSpawnFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"agent_nickname":   "*protocolv2.Nullable[string]",
		"agent_path":       "*protocolv2.Nullable[string]",
		"agent_role":       "*protocolv2.Nullable[string]",
		"depth":            "int32",
		"parent_thread_id": "string",
	} {
		if got := threadSpawnFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("SubAgentSource.thread_spawn.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	if !threadSpawnFields["depth"].Required || !threadSpawnFields["parent_thread_id"].Required {
		t.Fatalf("SubAgentSource.thread_spawn required fields = %#v", threadSpawnFields)
	}
	if otherSubAgent.DirectValueField == nil ||
		otherSubAgent.DirectValueField.FieldName != "other" ||
		otherSubAgent.DirectValueField.GoType != "string" {
		t.Fatalf("SubAgentSource other direct field = %#v", otherSubAgent.DirectValueField)
	}
}

func TestSelectGeneratedUntaggedObjectUnions(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	unions, err := SelectGeneratedUntaggedObjectUnions(plan)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(unions), 1; got != want {
		t.Fatalf("selected generated untagged object union count = %d, want %d", got, want)
	}
	union := unions[0]
	if union.TypeName != "ResourceContent" {
		t.Fatalf("untagged object union type = %q, want ResourceContent", union.TypeName)
	}
	if got, want := len(union.Variants), 2; got != want {
		t.Fatalf("ResourceContent variant count = %d, want %d", got, want)
	}
	var blob UntaggedObjectUnionVariantPlan
	var text UntaggedObjectUnionVariantPlan
	for _, variant := range union.Variants {
		switch variant.DiscriminatorValue {
		case "blob":
			blob = variant
		case "text":
			text = variant
		default:
			t.Fatalf("unexpected ResourceContent variant = %#v", variant)
		}
	}
	if blob.PayloadTypeName != "ResourceContentBlob" || blob.ConstructorName != "NewResourceContentBlob" {
		t.Fatalf("ResourceContent blob variant = %#v", blob)
	}
	if text.PayloadTypeName != "ResourceContentText" || text.ConstructorName != "NewResourceContentText" {
		t.Fatalf("ResourceContent text variant = %#v", text)
	}
	textFields := map[string]FieldPlan{}
	for _, field := range text.Fields {
		textFields[field.FieldName] = field
	}
	if textFields["_meta"].GoType != "*protocolv2.JSONValue" ||
		textFields["_meta"].Kind != FieldPlanJSONValue ||
		textFields["mimeType"].GoType != "*protocolv2.Nullable[string]" ||
		textFields["text"].GoType != "string" || !textFields["text"].Required ||
		textFields["uri"].GoType != "string" || !textFields["uri"].Required {
		t.Fatalf("ResourceContent text fields = %#v", textFields)
	}
	blobFields := map[string]FieldPlan{}
	for _, field := range blob.Fields {
		blobFields[field.FieldName] = field
	}
	if blobFields["blob"].GoType != "string" || !blobFields["blob"].Required ||
		blobFields["mimeType"].GoType != "*protocolv2.Nullable[string]" ||
		blobFields["uri"].GoType != "string" || !blobFields["uri"].Required {
		t.Fatalf("ResourceContent blob fields = %#v", blobFields)
	}
}

func TestReviewDecisionResponseDefinitionsStayShared(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	applyPatch, ok := plan.TypeBySchema("ApplyPatchApprovalResponse.json")
	if !ok {
		t.Fatal("missing ApplyPatchApprovalResponse schema")
	}
	execCommand, ok := plan.TypeBySchema("ExecCommandApprovalResponse.json")
	if !ok {
		t.Fatal("missing ExecCommandApprovalResponse schema")
	}
	for _, name := range []string{"ReviewDecision", "NetworkPolicyAmendment"} {
		left, err := json.Marshal(applyPatch.Schema.Definitions[name])
		if err != nil {
			t.Fatal(err)
		}
		right, err := json.Marshal(execCommand.Schema.Definitions[name])
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(left, right) {
			t.Fatalf("%s definition differs between ApplyPatchApprovalResponse and ExecCommandApprovalResponse", name)
		}
	}

	commandApproval, ok := plan.TypeBySchema("CommandExecutionRequestApprovalResponse.json")
	if !ok {
		t.Fatal("missing CommandExecutionRequestApprovalResponse schema")
	}
	left, err := json.Marshal(applyPatch.Schema.Definitions["NetworkPolicyAmendment"])
	if err != nil {
		t.Fatal(err)
	}
	right, err := json.Marshal(commandApproval.Schema.Definitions["NetworkPolicyAmendment"])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(left, right) {
		t.Fatal("NetworkPolicyAmendment definition differs between ApplyPatchApprovalResponse and CommandExecutionRequestApprovalResponse")
	}
}

func TestPermissionApprovalDefinitionsStayShared(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	commandApproval, ok := plan.TypeBySchema("CommandExecutionRequestApprovalParams.json")
	if !ok {
		t.Fatal("missing CommandExecutionRequestApprovalParams schema")
	}
	permissionsRequest, ok := plan.TypeBySchema("PermissionsRequestApprovalParams.json")
	if !ok {
		t.Fatal("missing PermissionsRequestApprovalParams schema")
	}
	permissionsResponse, ok := plan.TypeBySchema("PermissionsRequestApprovalResponse.json")
	if !ok {
		t.Fatal("missing PermissionsRequestApprovalResponse schema")
	}
	pathAliasName := "ApiPathString"
	if _, ok := commandApproval.Schema.Definitions[pathAliasName]; !ok {
		pathAliasName = "LegacyAppPathString"
	}
	for _, name := range []string{
		"AdditionalFileSystemPermissions",
		"AdditionalNetworkPermissions",
		pathAliasName,
		"FileSystemAccessMode",
		"FileSystemPath",
		"FileSystemSandboxEntry",
		"FileSystemSpecialPath",
	} {
		commandDefinition := encodedDefinition(t, commandApproval, name)
		requestDefinition := encodedDefinition(t, permissionsRequest, name)
		responseDefinition := encodedDefinition(t, permissionsResponse, name)
		if !bytes.Equal(commandDefinition, requestDefinition) {
			t.Fatalf("%s definition differs between CommandExecutionRequestApprovalParams and PermissionsRequestApprovalParams", name)
		}
		if !bytes.Equal(commandDefinition, responseDefinition) {
			t.Fatalf("%s definition differs between CommandExecutionRequestApprovalParams and PermissionsRequestApprovalResponse", name)
		}
	}
	if !bytes.Equal(encodedDefinition(t, commandApproval, "AbsolutePathBuf"), encodedDefinition(t, permissionsRequest, "AbsolutePathBuf")) {
		t.Fatal("AbsolutePathBuf definition differs between CommandExecutionRequestApprovalParams and PermissionsRequestApprovalParams")
	}
}

func TestGuardianApprovalReviewDefinitionsStayShared(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	started, ok := plan.TypeBySchema("v2/ItemGuardianApprovalReviewStartedNotification.json")
	if !ok {
		t.Fatal("missing ItemGuardianApprovalReviewStartedNotification schema")
	}
	completed, ok := plan.TypeBySchema("v2/ItemGuardianApprovalReviewCompletedNotification.json")
	if !ok {
		t.Fatal("missing ItemGuardianApprovalReviewCompletedNotification schema")
	}
	for _, name := range []string{
		"GuardianApprovalReview",
		"GuardianApprovalReviewAction",
		"RequestPermissionProfile",
	} {
		if !bytes.Equal(encodedDefinition(t, started, name), encodedDefinition(t, completed, name)) {
			t.Fatalf("%s definition differs between guardian approval review started and completed notifications", name)
		}
	}
}

func TestCommandExecPermissionDefinitionsStayShared(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	commandExec, ok := plan.TypeBySchema("v2/CommandExecParams.json")
	if !ok {
		t.Fatal("missing CommandExecParams schema")
	}
	commandApproval, ok := plan.TypeBySchema("CommandExecutionRequestApprovalParams.json")
	if !ok {
		t.Fatal("missing CommandExecutionRequestApprovalParams schema")
	}
	for _, name := range []string{"AbsolutePathBuf"} {
		if !bytes.Equal(encodedDefinition(t, commandExec, name), encodedDefinition(t, commandApproval, name)) {
			t.Fatalf("%s definition differs between CommandExecParams and CommandExecutionRequestApprovalParams", name)
		}
	}
}

func TestAccountRateLimitDefinitionsStayShared(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	response, ok := plan.TypeBySchema("v2/GetAccountRateLimitsResponse.json")
	if !ok {
		t.Fatal("missing v2/GetAccountRateLimitsResponse schema")
	}
	notification, ok := plan.TypeBySchema("v2/AccountRateLimitsUpdatedNotification.json")
	if !ok {
		t.Fatal("missing v2/AccountRateLimitsUpdatedNotification schema")
	}
	for _, name := range []string{
		"CreditsSnapshot",
		"PlanType",
		"RateLimitReachedType",
		"RateLimitSnapshot",
		"RateLimitWindow",
	} {
		if !bytes.Equal(encodedDefinition(t, response, name), encodedDefinition(t, notification, name)) {
			t.Fatalf("%s definition differs between account rate limit response and notification", name)
		}
	}
}

func TestThreadTurnParamDefinitionsStayShared(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	threadStart, ok := plan.TypeBySchema("v2/ThreadStartParams.json")
	if !ok {
		t.Fatal("missing v2/ThreadStartParams schema")
	}
	threadFork, ok := plan.TypeBySchema("v2/ThreadForkParams.json")
	if !ok {
		t.Fatal("missing v2/ThreadForkParams schema")
	}
	turnStart, ok := plan.TypeBySchema("v2/TurnStartParams.json")
	if !ok {
		t.Fatal("missing v2/TurnStartParams schema")
	}
	for _, name := range []string{
		"AbsolutePathBuf",
		"ApprovalsReviewer",
		"AskForApproval",
	} {
		startDefinition := encodedDefinition(t, threadStart, name)
		forkDefinition := encodedDefinition(t, threadFork, name)
		turnDefinition := encodedDefinition(t, turnStart, name)
		if !bytes.Equal(startDefinition, forkDefinition) {
			t.Fatalf("%s definition differs between ThreadStartParams and ThreadForkParams", name)
		}
		if !bytes.Equal(startDefinition, turnDefinition) {
			t.Fatalf("%s definition differs between ThreadStartParams and TurnStartParams", name)
		}
	}
	for _, name := range []string{"Personality", "TurnEnvironmentParams"} {
		if !bytes.Equal(encodedDefinition(t, threadStart, name), encodedDefinition(t, turnStart, name)) {
			t.Fatalf("%s definition differs between ThreadStartParams and TurnStartParams", name)
		}
	}
	if !bytes.Equal(encodedDefinition(t, threadStart, "ThreadSource"), encodedDefinition(t, threadFork, "ThreadSource")) {
		t.Fatal("ThreadSource definition differs between ThreadStartParams and ThreadForkParams")
	}
	if !bytes.Equal(encodedDefinition(t, threadStart, "SandboxMode"), encodedDefinition(t, threadFork, "SandboxMode")) {
		t.Fatal("SandboxMode definition differs between ThreadStartParams and ThreadForkParams")
	}
}

func encodedDefinition(t *testing.T, typ TypePlan, name string) []byte {
	t.Helper()
	if typ.Schema == nil || typ.Schema.Definitions[name] == nil {
		t.Fatalf("%s definition is missing from %s", name, typ.SchemaPath)
	}
	raw, err := json.Marshal(typ.Schema.Definitions[name])
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestSelectGeneratedTaggedUnions(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	unions, err := SelectGeneratedTaggedUnions(plan)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(unions), 36; got != want {
		t.Fatalf("selected generated tagged union count = %d, want %d", got, want)
	}
	unionByName := map[string]TaggedUnionPlan{}
	for _, union := range unions {
		unionByName[union.TypeName] = union
	}
	configuredHookHandler := unionByName["ConfiguredHookHandler"]
	if configuredHookHandler.Discriminator != "type" {
		t.Fatalf("ConfiguredHookHandler discriminator = %q, want type", configuredHookHandler.Discriminator)
	}
	if got, want := len(configuredHookHandler.Variants), 3; got != want {
		t.Fatalf("ConfiguredHookHandler variant count = %d, want %d", got, want)
	}
	var commandHook TaggedUnionVariantPlan
	var promptHook TaggedUnionVariantPlan
	var agentHook TaggedUnionVariantPlan
	for _, variant := range configuredHookHandler.Variants {
		switch variant.DiscriminatorValue {
		case "command":
			commandHook = variant
		case "prompt":
			promptHook = variant
		case "agent":
			agentHook = variant
		}
	}
	if commandHook.PayloadTypeName != "ConfiguredHookHandlerCommand" || commandHook.ConstructorName != "NewConfiguredHookHandlerCommand" {
		t.Fatalf("ConfiguredHookHandler command variant = %#v", commandHook)
	}
	commandHookFields := map[string]FieldPlan{}
	for _, field := range commandHook.Fields {
		commandHookFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"async":         "bool",
		"command":       "string",
		"statusMessage": "*protocolv2.Nullable[string]",
		"timeoutSec":    "*protocolv2.Nullable[uint64]",
	} {
		if got := commandHookFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("ConfiguredHookHandler.command.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}
	if promptHook.PayloadTypeName != "ConfiguredHookHandlerPrompt" || len(promptHook.Fields) != 0 {
		t.Fatalf("ConfiguredHookHandler prompt variant = %#v", promptHook)
	}
	if agentHook.PayloadTypeName != "ConfiguredHookHandlerAgent" || len(agentHook.Fields) != 0 {
		t.Fatalf("ConfiguredHookHandler agent variant = %#v", agentHook)
	}

	dynamicToolSpec := unionByName["DynamicToolSpec"]
	if dynamicToolSpec.Discriminator != "type" {
		t.Fatalf("DynamicToolSpec discriminator = %q, want type", dynamicToolSpec.Discriminator)
	}
	if got, want := len(dynamicToolSpec.Variants), 2; got != want {
		t.Fatalf("DynamicToolSpec variant count = %d, want %d", got, want)
	}
	var dynamicToolFunction TaggedUnionVariantPlan
	var dynamicToolNamespace TaggedUnionVariantPlan
	for _, variant := range dynamicToolSpec.Variants {
		switch variant.DiscriminatorValue {
		case "function":
			dynamicToolFunction = variant
		case "namespace":
			dynamicToolNamespace = variant
		}
	}
	dynamicToolFunctionFields := map[string]FieldPlan{}
	for _, field := range dynamicToolFunction.Fields {
		dynamicToolFunctionFields[field.FieldName] = field
	}
	if dynamicToolFunction.PayloadTypeName != "DynamicToolSpecFunction" ||
		dynamicToolFunctionFields["description"].GoType != "string" ||
		dynamicToolFunctionFields["inputSchema"].Kind != FieldPlanJSONValue ||
		dynamicToolFunctionFields["name"].GoType != "string" {
		t.Fatalf("DynamicToolSpec function variant = %#v", dynamicToolFunction)
	}
	dynamicToolNamespaceFields := map[string]FieldPlan{}
	for _, field := range dynamicToolNamespace.Fields {
		dynamicToolNamespaceFields[field.FieldName] = field
	}
	if dynamicToolNamespace.PayloadTypeName != "DynamicToolSpecNamespace" ||
		dynamicToolNamespaceFields["description"].GoType != "string" ||
		dynamicToolNamespaceFields["name"].GoType != "string" ||
		dynamicToolNamespaceFields["tools"].GoType != "[]DynamicToolNamespaceTool" {
		t.Fatalf("DynamicToolSpec namespace variant = %#v", dynamicToolNamespace)
	}

	dynamicToolNamespaceTool := unionByName["DynamicToolNamespaceTool"]
	if dynamicToolNamespaceTool.Discriminator != "type" {
		t.Fatalf("DynamicToolNamespaceTool discriminator = %q, want type", dynamicToolNamespaceTool.Discriminator)
	}
	if got, want := len(dynamicToolNamespaceTool.Variants), 1; got != want {
		t.Fatalf("DynamicToolNamespaceTool variant count = %d, want %d", got, want)
	}
	namespaceToolFunction := dynamicToolNamespaceTool.Variants[0]
	namespaceToolFunctionFields := map[string]FieldPlan{}
	for _, field := range namespaceToolFunction.Fields {
		namespaceToolFunctionFields[field.FieldName] = field
	}
	if namespaceToolFunction.DiscriminatorValue != "function" ||
		namespaceToolFunction.PayloadTypeName != "DynamicToolNamespaceToolFunction" ||
		namespaceToolFunctionFields["description"].GoType != "string" ||
		namespaceToolFunctionFields["inputSchema"].Kind != FieldPlanJSONValue ||
		namespaceToolFunctionFields["name"].GoType != "string" {
		t.Fatalf("DynamicToolNamespaceTool function variant = %#v", namespaceToolFunction)
	}

	notification := unionByName["ClientNotification"]
	if notification.Discriminator != "method" {
		t.Fatalf("ClientNotification discriminator = %q, want method", notification.Discriminator)
	}
	if got, want := len(notification.Variants), 1; got != want {
		t.Fatalf("ClientNotification variant count = %d, want %d", got, want)
	}
	if notification.Variants[0].DiscriminatorValue != "initialized" || len(notification.Variants[0].Fields) != 0 {
		t.Fatalf("ClientNotification variant = %#v", notification.Variants[0])
	}

	clientRequest := unionByName["ClientRequest"]
	if clientRequest.Discriminator != "method" {
		t.Fatalf("ClientRequest discriminator = %q, want method", clientRequest.Discriminator)
	}
	if got, min := len(clientRequest.Variants), 119; got < min {
		t.Fatalf("ClientRequest variant count = %d, want at least %d", got, min)
	}
	var threadStartRequest TaggedUnionVariantPlan
	var memoryResetRequest TaggedUnionVariantPlan
	for _, variant := range clientRequest.Variants {
		switch variant.DiscriminatorValue {
		case "thread/start":
			threadStartRequest = variant
		case "memory/reset":
			memoryResetRequest = variant
		}
	}
	threadStartFields := map[string]FieldPlan{}
	for _, field := range threadStartRequest.Fields {
		threadStartFields[field.FieldName] = field
	}
	if threadStartRequest.PayloadTypeName != "ClientRequestThreadStart" ||
		threadStartFields["id"].GoType != "RequestId" ||
		threadStartFields["params"].GoType != "ThreadStartParams" ||
		!threadStartFields["id"].Required ||
		!threadStartFields["params"].Required {
		t.Fatalf("ClientRequest thread/start variant = %#v", threadStartRequest)
	}
	memoryResetFields := map[string]FieldPlan{}
	for _, field := range memoryResetRequest.Fields {
		memoryResetFields[field.FieldName] = field
	}
	if memoryResetRequest.PayloadTypeName != "ClientRequestMemoryReset" ||
		memoryResetFields["id"].GoType != "RequestId" ||
		!memoryResetFields["id"].Required ||
		len(memoryResetFields) != 1 ||
		len(memoryResetRequest.NullFields) != 1 ||
		memoryResetRequest.NullFields[0].FieldName != "params" {
		t.Fatalf("ClientRequest memory/reset variant = %#v", memoryResetRequest)
	}

	serverNotification := unionByName["ServerNotification"]
	if serverNotification.Discriminator != "method" {
		t.Fatalf("ServerNotification discriminator = %q, want method", serverNotification.Discriminator)
	}
	if got, min := len(serverNotification.Variants), 66; got < min {
		t.Fatalf("ServerNotification variant count = %d, want at least %d", got, min)
	}
	var errorNotification TaggedUnionVariantPlan
	var tokenUsageNotification TaggedUnionVariantPlan
	var realtimeSDPNotification TaggedUnionVariantPlan
	for _, variant := range serverNotification.Variants {
		switch variant.DiscriminatorValue {
		case "error":
			errorNotification = variant
		case "thread/tokenUsage/updated":
			tokenUsageNotification = variant
		case "thread/realtime/sdp":
			realtimeSDPNotification = variant
		}
	}
	for name, variant := range map[string]TaggedUnionVariantPlan{
		"error":                     errorNotification,
		"thread/tokenUsage/updated": tokenUsageNotification,
		"thread/realtime/sdp":       realtimeSDPNotification,
	} {
		fields := map[string]FieldPlan{}
		for _, field := range variant.Fields {
			fields[field.FieldName] = field
		}
		if fields["params"].Kind != FieldPlanRef || !fields["params"].Required {
			t.Fatalf("ServerNotification %s params field = %#v", name, fields["params"])
		}
	}
	if errorNotification.PayloadTypeName != "ServerNotificationError" ||
		errorNotification.Fields[0].GoType != "ErrorNotification" ||
		tokenUsageNotification.PayloadTypeName != "ServerNotificationThreadTokenUsageUpdated" ||
		tokenUsageNotification.Fields[0].GoType != "ThreadTokenUsageUpdatedNotification" ||
		realtimeSDPNotification.PayloadTypeName != "ServerNotificationThreadRealtimeSDP" ||
		realtimeSDPNotification.Fields[0].GoType != "ThreadRealtimeSdpNotification" {
		t.Fatalf("ServerNotification variants = %#v %#v %#v", errorNotification, tokenUsageNotification, realtimeSDPNotification)
	}

	serverRequest := unionByName["ServerRequest"]
	if serverRequest.Discriminator != "method" {
		t.Fatalf("ServerRequest discriminator = %q, want method", serverRequest.Discriminator)
	}
	if got, min := len(serverRequest.Variants), 10; got < min {
		t.Fatalf("ServerRequest variant count = %d, want at least %d", got, min)
	}
	var commandApprovalRequest TaggedUnionVariantPlan
	for _, variant := range serverRequest.Variants {
		if variant.DiscriminatorValue == "item/commandExecution/requestApproval" {
			commandApprovalRequest = variant
		}
	}
	commandApprovalFields := map[string]FieldPlan{}
	for _, field := range commandApprovalRequest.Fields {
		commandApprovalFields[field.FieldName] = field
	}
	if commandApprovalRequest.PayloadTypeName != "ServerRequestItemCommandExecutionRequestApproval" ||
		commandApprovalFields["id"].GoType != "RequestId" ||
		commandApprovalFields["params"].GoType != "CommandExecutionRequestApprovalParams" ||
		!commandApprovalFields["id"].Required ||
		!commandApprovalFields["params"].Required {
		t.Fatalf("ServerRequest item/commandExecution/requestApproval variant = %#v", commandApprovalRequest)
	}

	realtimeTransport := unionByName["ThreadRealtimeStartTransport"]
	if realtimeTransport.Discriminator != "type" {
		t.Fatalf("ThreadRealtimeStartTransport discriminator = %q, want type", realtimeTransport.Discriminator)
	}
	if got, want := len(realtimeTransport.Variants), 2; got != want {
		t.Fatalf("ThreadRealtimeStartTransport variant count = %d, want %d", got, want)
	}
	var websocketTransport TaggedUnionVariantPlan
	var webrtcTransport TaggedUnionVariantPlan
	for _, variant := range realtimeTransport.Variants {
		switch variant.DiscriminatorValue {
		case "websocket":
			websocketTransport = variant
		case "webrtc":
			webrtcTransport = variant
		}
	}
	webrtcFields := map[string]FieldPlan{}
	for _, field := range webrtcTransport.Fields {
		webrtcFields[field.FieldName] = field
	}
	if websocketTransport.PayloadTypeName != "ThreadRealtimeStartTransportWebsocket" ||
		len(websocketTransport.Fields) != 0 ||
		webrtcTransport.PayloadTypeName != "ThreadRealtimeStartTransportWebrtc" ||
		webrtcFields["sdp"].GoType != "string" ||
		!webrtcFields["sdp"].Required {
		t.Fatalf("ThreadRealtimeStartTransport variants = %#v %#v", websocketTransport, webrtcTransport)
	}

	params := unionByName["LoginAccountParams"]
	if params.Discriminator != "type" {
		t.Fatalf("LoginAccountParams discriminator = %q, want type", params.Discriminator)
	}
	if got, want := len(params.Variants), 4; got != want {
		t.Fatalf("LoginAccountParams variant count = %d, want %d", got, want)
	}
	var apiKeyVariant TaggedUnionVariantPlan
	for _, variant := range params.Variants {
		if variant.DiscriminatorValue == "apiKey" {
			apiKeyVariant = variant
		}
	}
	if apiKeyVariant.ConstructorName != "NewLoginAccountParamsAPIKey" {
		t.Fatalf("apiKey constructor = %q", apiKeyVariant.ConstructorName)
	}
	if apiKeyVariant.AccessorName != "AsAPIKey" {
		t.Fatalf("apiKey accessor = %q", apiKeyVariant.AccessorName)
	}
	if got, want := len(apiKeyVariant.Fields), 1; got != want {
		t.Fatalf("apiKey field count = %d, want %d", got, want)
	}
	if apiKeyVariant.Fields[0].FieldName != "apiKey" || apiKeyVariant.Fields[0].GoType != "string" {
		t.Fatalf("apiKey field = %#v", apiKeyVariant.Fields[0])
	}

	account := unionByName["Account"]
	if account.Discriminator != "type" {
		t.Fatalf("Account discriminator = %q, want type", account.Discriminator)
	}
	if got, want := len(account.Variants), 3; got != want {
		t.Fatalf("Account variant count = %d, want %d", got, want)
	}
	var chatGPTAccount TaggedUnionVariantPlan
	for _, variant := range account.Variants {
		if variant.DiscriminatorValue == "chatgpt" {
			chatGPTAccount = variant
		}
	}
	if chatGPTAccount.ConstructorName != "NewAccountChatGPT" {
		t.Fatalf("chatgpt account constructor = %q", chatGPTAccount.ConstructorName)
	}
	if got, want := len(chatGPTAccount.Fields), 2; got != want {
		t.Fatalf("chatgpt account field count = %d, want %d", got, want)
	}
	accountFields := map[string]FieldPlan{}
	for _, field := range chatGPTAccount.Fields {
		accountFields[field.FieldName] = field
	}
	emailField := accountFields["email"]
	if (emailField.GoType != "string" && emailField.GoType != "protocolv2.Nullable[string]") ||
		!emailField.Required ||
		accountFields["planType"].GoType != "PlanType" {
		t.Fatalf("chatgpt account fields = %#v", accountFields)
	}

	fileChange := unionByName["FileChange"]
	if fileChange.Discriminator != "type" {
		t.Fatalf("FileChange discriminator = %q, want type", fileChange.Discriminator)
	}
	if got, want := len(fileChange.Variants), 3; got != want {
		t.Fatalf("FileChange variant count = %d, want %d", got, want)
	}
	var updateVariant TaggedUnionVariantPlan
	for _, variant := range fileChange.Variants {
		if variant.DiscriminatorValue == "update" {
			updateVariant = variant
		}
	}
	if updateVariant.PayloadTypeName != "FileChangeUpdate" {
		t.Fatalf("FileChange update payload = %q, want FileChangeUpdate", updateVariant.PayloadTypeName)
	}
	if got, want := len(updateVariant.Fields), 2; got != want {
		t.Fatalf("FileChange update field count = %d, want %d", got, want)
	}

	parsedCommand := unionByName["ParsedCommand"]
	if parsedCommand.Discriminator != "type" {
		t.Fatalf("ParsedCommand discriminator = %q, want type", parsedCommand.Discriminator)
	}
	if got, want := len(parsedCommand.Variants), 4; got != want {
		t.Fatalf("ParsedCommand variant count = %d, want %d", got, want)
	}
	var listFilesVariant TaggedUnionVariantPlan
	for _, variant := range parsedCommand.Variants {
		if variant.DiscriminatorValue == "list_files" {
			listFilesVariant = variant
		}
	}
	if listFilesVariant.PayloadTypeName != "ParsedCommandListFiles" {
		t.Fatalf("ParsedCommand list_files payload = %q, want ParsedCommandListFiles", listFilesVariant.PayloadTypeName)
	}

	dynamicContent := unionByName["DynamicToolCallOutputContentItem"]
	if dynamicContent.Discriminator != "type" {
		t.Fatalf("DynamicToolCallOutputContentItem discriminator = %q, want type", dynamicContent.Discriminator)
	}
	if got, want := len(dynamicContent.Variants), 2; got != want {
		t.Fatalf("DynamicToolCallOutputContentItem variant count = %d, want %d", got, want)
	}
	var inputImageVariant TaggedUnionVariantPlan
	for _, variant := range dynamicContent.Variants {
		if variant.DiscriminatorValue == "inputImage" {
			inputImageVariant = variant
		}
	}
	if inputImageVariant.PayloadTypeName != "DynamicToolCallOutputContentItemInputImage" {
		t.Fatalf("DynamicToolCallOutputContentItem inputImage payload = %q, want DynamicToolCallOutputContentItemInputImage", inputImageVariant.PayloadTypeName)
	}
	if got, want := len(inputImageVariant.Fields), 1; got != want {
		t.Fatalf("DynamicToolCallOutputContentItem inputImage field count = %d, want %d", got, want)
	}

	contentItem := unionByName["ContentItem"]
	if contentItem.Discriminator != "type" {
		t.Fatalf("ContentItem discriminator = %q, want type", contentItem.Discriminator)
	}
	if got, want := len(contentItem.Variants), 3; got != want {
		t.Fatalf("ContentItem variant count = %d, want %d", got, want)
	}
	functionOutputContent := unionByName["FunctionCallOutputContentItem"]
	if functionOutputContent.Discriminator != "type" {
		t.Fatalf("FunctionCallOutputContentItem discriminator = %q, want type", functionOutputContent.Discriminator)
	}
	if got, want := len(functionOutputContent.Variants), 3; got != want {
		t.Fatalf("FunctionCallOutputContentItem variant count = %d, want %d", got, want)
	}
	localShellAction := unionByName["LocalShellAction"]
	if localShellAction.Discriminator != "type" {
		t.Fatalf("LocalShellAction discriminator = %q, want type", localShellAction.Discriminator)
	}
	if got, want := len(localShellAction.Variants), 1; got != want {
		t.Fatalf("LocalShellAction variant count = %d, want %d", got, want)
	}
	reasoningContent := unionByName["ReasoningItemContent"]
	if reasoningContent.Discriminator != "type" {
		t.Fatalf("ReasoningItemContent discriminator = %q, want type", reasoningContent.Discriminator)
	}
	if got, want := len(reasoningContent.Variants), 2; got != want {
		t.Fatalf("ReasoningItemContent variant count = %d, want %d", got, want)
	}
	reasoningSummary := unionByName["ReasoningItemReasoningSummary"]
	if reasoningSummary.Discriminator != "type" {
		t.Fatalf("ReasoningItemReasoningSummary discriminator = %q, want type", reasoningSummary.Discriminator)
	}
	if got, want := len(reasoningSummary.Variants), 1; got != want {
		t.Fatalf("ReasoningItemReasoningSummary variant count = %d, want %d", got, want)
	}
	webSearchAction := unionByName["ResponsesApiWebSearchAction"]
	if webSearchAction.Discriminator != "type" {
		t.Fatalf("ResponsesApiWebSearchAction discriminator = %q, want type", webSearchAction.Discriminator)
	}
	if got, want := len(webSearchAction.Variants), 4; got != want {
		t.Fatalf("ResponsesApiWebSearchAction variant count = %d, want %d", got, want)
	}

	responseItem := unionByName["ResponseItem"]
	if responseItem.Discriminator != "type" {
		t.Fatalf("ResponseItem discriminator = %q, want type", responseItem.Discriminator)
	}
	if got, want := len(responseItem.Variants), 16; got != want {
		t.Fatalf("ResponseItem variant count = %d, want %d", got, want)
	}
	var messageItem TaggedUnionVariantPlan
	var toolSearchCall TaggedUnionVariantPlan
	var functionCallOutput TaggedUnionVariantPlan
	var toolSearchOutput TaggedUnionVariantPlan
	for _, variant := range responseItem.Variants {
		switch variant.DiscriminatorValue {
		case "message":
			messageItem = variant
		case "tool_search_call":
			toolSearchCall = variant
		case "function_call_output":
			functionCallOutput = variant
		case "tool_search_output":
			toolSearchOutput = variant
		}
	}
	messageItemFields := map[string]FieldPlan{}
	for _, field := range messageItem.Fields {
		messageItemFields[field.FieldName] = field
	}
	if messageItemFields["content"].GoType != "[]ContentItem" || messageItemFields["phase"].GoType != "*protocolv2.Nullable[MessagePhase]" {
		t.Fatalf("ResponseItem.message fields = %#v", messageItemFields)
	}
	toolSearchCallFields := map[string]FieldPlan{}
	for _, field := range toolSearchCall.Fields {
		toolSearchCallFields[field.FieldName] = field
	}
	if toolSearchCallFields["arguments"].GoType != "protocolv2.JSONValue" || toolSearchCallFields["arguments"].Kind != FieldPlanJSONValue {
		t.Fatalf("ResponseItem.tool_search_call arguments field = %#v", toolSearchCallFields["arguments"])
	}
	functionCallOutputFields := map[string]FieldPlan{}
	for _, field := range functionCallOutput.Fields {
		functionCallOutputFields[field.FieldName] = field
	}
	if functionCallOutputFields["output"].GoType != "FunctionCallOutputBody" {
		t.Fatalf("ResponseItem.function_call_output fields = %#v", functionCallOutputFields)
	}
	toolSearchOutputFields := map[string]FieldPlan{}
	for _, field := range toolSearchOutput.Fields {
		toolSearchOutputFields[field.FieldName] = field
	}
	if toolSearchOutputFields["tools"].GoType != "[]protocolv2.JSONValue" || toolSearchOutputFields["tools"].Kind != FieldPlanArrayJSONValue {
		t.Fatalf("ResponseItem.tool_search_output tools field = %#v", toolSearchOutputFields["tools"])
	}

	commandAction := unionByName["CommandAction"]
	if commandAction.Discriminator != "type" {
		t.Fatalf("CommandAction discriminator = %q, want type", commandAction.Discriminator)
	}
	if got, want := len(commandAction.Variants), 4; got != want {
		t.Fatalf("CommandAction variant count = %d, want %d", got, want)
	}
	var commandActionRead TaggedUnionVariantPlan
	for _, variant := range commandAction.Variants {
		if variant.DiscriminatorValue == "read" {
			commandActionRead = variant
		}
	}
	if commandActionRead.PayloadTypeName != "CommandActionRead" {
		t.Fatalf("CommandAction read payload = %q, want CommandActionRead", commandActionRead.PayloadTypeName)
	}
	if got, want := len(commandActionRead.Fields), 3; got != want {
		t.Fatalf("CommandAction read field count = %d, want %d", got, want)
	}

	fileSystemPath := unionByName["FileSystemPath"]
	if fileSystemPath.Discriminator != "type" {
		t.Fatalf("FileSystemPath discriminator = %q, want type", fileSystemPath.Discriminator)
	}
	if got, want := len(fileSystemPath.Variants), 3; got != want {
		t.Fatalf("FileSystemPath variant count = %d, want %d", got, want)
	}
	var literalPath TaggedUnionVariantPlan
	var specialPath TaggedUnionVariantPlan
	for _, variant := range fileSystemPath.Variants {
		if variant.DiscriminatorValue == "path" {
			literalPath = variant
		}
		if variant.DiscriminatorValue == "special" {
			specialPath = variant
		}
	}
	if literalPath.PayloadTypeName != "FileSystemPathPath" {
		t.Fatalf("FileSystemPath path payload = %q, want FileSystemPathPath", literalPath.PayloadTypeName)
	}
	if got, want := len(literalPath.Fields), 1; got != want {
		t.Fatalf("FileSystemPath path field count = %d, want %d", got, want)
	}
	if literalPath.Fields[0].FieldName != "path" || literalPath.Fields[0].GoType != "string" || literalPath.Fields[0].Kind != FieldPlanScalar {
		t.Fatalf("FileSystemPath path field = %#v", literalPath.Fields[0])
	}
	if specialPath.PayloadTypeName != "FileSystemPathSpecial" {
		t.Fatalf("FileSystemPath special payload = %q, want FileSystemPathSpecial", specialPath.PayloadTypeName)
	}
	if got, want := len(specialPath.Fields), 1; got != want {
		t.Fatalf("FileSystemPath special field count = %d, want %d", got, want)
	}
	if specialPath.Fields[0].FieldName != "value" || specialPath.Fields[0].GoType != "FileSystemSpecialPath" {
		t.Fatalf("FileSystemPath special field = %#v", specialPath.Fields[0])
	}

	fileSystemSpecialPath := unionByName["FileSystemSpecialPath"]
	if fileSystemSpecialPath.Discriminator != "kind" {
		t.Fatalf("FileSystemSpecialPath discriminator = %q, want kind", fileSystemSpecialPath.Discriminator)
	}
	if got, want := len(fileSystemSpecialPath.Variants), 6; got != want {
		t.Fatalf("FileSystemSpecialPath variant count = %d, want %d", got, want)
	}

	guardianAction := unionByName["GuardianApprovalReviewAction"]
	if guardianAction.Discriminator != "type" {
		t.Fatalf("GuardianApprovalReviewAction discriminator = %q, want type", guardianAction.Discriminator)
	}
	if got, want := len(guardianAction.Variants), 6; got != want {
		t.Fatalf("GuardianApprovalReviewAction variant count = %d, want %d", got, want)
	}
	guardianActionByValue := map[string]TaggedUnionVariantPlan{}
	for _, variant := range guardianAction.Variants {
		guardianActionByValue[variant.DiscriminatorValue] = variant
	}
	for value, wantPayload := range map[string]string{
		"command":            "GuardianApprovalReviewActionCommand",
		"execve":             "GuardianApprovalReviewActionExecve",
		"applyPatch":         "GuardianApprovalReviewActionApplyPatch",
		"networkAccess":      "GuardianApprovalReviewActionNetworkAccess",
		"mcpToolCall":        "GuardianApprovalReviewActionMCPToolCall",
		"requestPermissions": "GuardianApprovalReviewActionRequestPermissions",
	} {
		if guardianActionByValue[value].PayloadTypeName != wantPayload {
			t.Fatalf("GuardianApprovalReviewAction %s payload = %#v, want %s", value, guardianActionByValue[value], wantPayload)
		}
	}
	guardianCommandFields := map[string]FieldPlan{}
	for _, field := range guardianActionByValue["command"].Fields {
		guardianCommandFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"command": "string",
		"cwd":     "string",
		"source":  "GuardianCommandSource",
	} {
		if guardianCommandFields[fieldName].GoType != wantGoType || !guardianCommandFields[fieldName].Required {
			t.Fatalf("GuardianApprovalReviewAction.command.%s field = %#v", fieldName, guardianCommandFields[fieldName])
		}
	}
	guardianNetworkFields := map[string]FieldPlan{}
	for _, field := range guardianActionByValue["networkAccess"].Fields {
		guardianNetworkFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"host":     "string",
		"port":     "uint16",
		"protocol": "NetworkApprovalProtocol",
		"target":   "string",
	} {
		if guardianNetworkFields[fieldName].GoType != wantGoType || !guardianNetworkFields[fieldName].Required {
			t.Fatalf("GuardianApprovalReviewAction.networkAccess.%s field = %#v", fieldName, guardianNetworkFields[fieldName])
		}
	}
	guardianRequestPermissionsFields := map[string]FieldPlan{}
	for _, field := range guardianActionByValue["requestPermissions"].Fields {
		guardianRequestPermissionsFields[field.FieldName] = field
	}
	if guardianRequestPermissionsFields["permissions"].GoType != "RequestPermissionProfile" ||
		!guardianRequestPermissionsFields["permissions"].Required ||
		guardianRequestPermissionsFields["reason"].GoType != "*protocolv2.Nullable[string]" ||
		guardianRequestPermissionsFields["reason"].Required {
		t.Fatalf("GuardianApprovalReviewAction.requestPermissions fields = %#v", guardianRequestPermissionsFields)
	}

	sandboxPolicy := unionByName["SandboxPolicy"]
	if sandboxPolicy.Discriminator != "type" {
		t.Fatalf("SandboxPolicy discriminator = %q, want type", sandboxPolicy.Discriminator)
	}
	if got, want := len(sandboxPolicy.Variants), 4; got != want {
		t.Fatalf("SandboxPolicy variant count = %d, want %d", got, want)
	}
	var externalSandbox TaggedUnionVariantPlan
	var workspaceWrite TaggedUnionVariantPlan
	for _, variant := range sandboxPolicy.Variants {
		switch variant.DiscriminatorValue {
		case "externalSandbox":
			externalSandbox = variant
		case "workspaceWrite":
			workspaceWrite = variant
		}
	}
	externalSandboxFields := map[string]FieldPlan{}
	for _, field := range externalSandbox.Fields {
		externalSandboxFields[field.FieldName] = field
	}
	if externalSandboxFields["networkAccess"].GoType != "*NetworkAccess" {
		t.Fatalf("SandboxPolicy externalSandbox fields = %#v", externalSandboxFields)
	}
	workspaceWriteFields := map[string]FieldPlan{}
	for _, field := range workspaceWrite.Fields {
		workspaceWriteFields[field.FieldName] = field
	}
	for fieldName, wantGoType := range map[string]string{
		"excludeSlashTmp":     "*bool",
		"excludeTmpdirEnvVar": "*bool",
		"networkAccess":       "*bool",
		"writableRoots":       "*[]string",
	} {
		if got := workspaceWriteFields[fieldName].GoType; got != wantGoType {
			t.Fatalf("SandboxPolicy.workspaceWrite.%s GoType = %q, want %q", fieldName, got, wantGoType)
		}
	}

	configLayerSource := unionByName["ConfigLayerSource"]
	if configLayerSource.Discriminator != "type" {
		t.Fatalf("ConfigLayerSource discriminator = %q, want type", configLayerSource.Discriminator)
	}
	if got, want := len(configLayerSource.Variants), 8; got != want {
		t.Fatalf("ConfigLayerSource variant count = %d, want %d", got, want)
	}
	var mdmLayer TaggedUnionVariantPlan
	var systemLayer TaggedUnionVariantPlan
	var projectLayer TaggedUnionVariantPlan
	var sessionFlagsLayer TaggedUnionVariantPlan
	for _, variant := range configLayerSource.Variants {
		switch variant.DiscriminatorValue {
		case "mdm":
			mdmLayer = variant
		case "system":
			systemLayer = variant
		case "project":
			projectLayer = variant
		case "sessionFlags":
			sessionFlagsLayer = variant
		}
	}
	if mdmLayer.ConstructorName != "NewConfigLayerSourceMdm" || len(mdmLayer.Fields) != 2 {
		t.Fatalf("ConfigLayerSource mdm variant = %#v", mdmLayer)
	}
	mdmLayerFields := map[string]FieldPlan{}
	for _, field := range mdmLayer.Fields {
		mdmLayerFields[field.FieldName] = field
	}
	if mdmLayerFields["domain"].GoType != "string" || mdmLayerFields["key"].GoType != "string" {
		t.Fatalf("ConfigLayerSource mdm fields = %#v", mdmLayerFields)
	}
	if systemLayer.PayloadTypeName != "ConfigLayerSourceSystem" || systemLayer.Fields[0].GoType != "string" {
		t.Fatalf("ConfigLayerSource system variant = %#v", systemLayer)
	}
	if projectLayer.PayloadTypeName != "ConfigLayerSourceProject" || projectLayer.Fields[0].FieldName != "dotCodexFolder" || projectLayer.Fields[0].GoType != "string" {
		t.Fatalf("ConfigLayerSource project variant = %#v", projectLayer)
	}
	if sessionFlagsLayer.PayloadTypeName != "ConfigLayerSourceSessionFlags" || len(sessionFlagsLayer.Fields) != 0 {
		t.Fatalf("ConfigLayerSource sessionFlags variant = %#v", sessionFlagsLayer)
	}

	userInput := unionByName["UserInput"]
	if userInput.Discriminator != "type" {
		t.Fatalf("UserInput discriminator = %q, want type", userInput.Discriminator)
	}
	if got, want := len(userInput.Variants), 5; got != want {
		t.Fatalf("UserInput variant count = %d, want %d", got, want)
	}
	var textInput TaggedUnionVariantPlan
	var imageInput TaggedUnionVariantPlan
	var mentionInput TaggedUnionVariantPlan
	for _, variant := range userInput.Variants {
		switch variant.DiscriminatorValue {
		case "text":
			textInput = variant
		case "image":
			imageInput = variant
		case "mention":
			mentionInput = variant
		}
	}
	if textInput.PayloadTypeName != "UserInputText" {
		t.Fatalf("UserInput text payload = %#v", textInput)
	}
	textInputFields := map[string]FieldPlan{}
	for _, field := range textInput.Fields {
		textInputFields[field.FieldName] = field
	}
	if textInputFields["text"].GoType != "string" || textInputFields["text_elements"].GoType != "*[]TextElement" {
		t.Fatalf("UserInput text fields = %#v", textInputFields)
	}
	imageInputFields := map[string]FieldPlan{}
	for _, field := range imageInput.Fields {
		imageInputFields[field.FieldName] = field
	}
	if imageInput.PayloadTypeName != "UserInputImage" ||
		imageInputFields["detail"].GoType != "*protocolv2.Nullable[ImageDetail]" ||
		imageInputFields["url"].GoType != "string" {
		t.Fatalf("UserInput image variant = %#v", imageInput)
	}
	mentionFields := map[string]FieldPlan{}
	for _, field := range mentionInput.Fields {
		mentionFields[field.FieldName] = field
	}
	if mentionFields["name"].GoType != "string" || mentionFields["path"].GoType != "string" {
		t.Fatalf("UserInput mention fields = %#v", mentionFields)
	}

	patchChangeKind := unionByName["PatchChangeKind"]
	if patchChangeKind.Discriminator != "type" {
		t.Fatalf("PatchChangeKind discriminator = %q, want type", patchChangeKind.Discriminator)
	}
	if got, want := len(patchChangeKind.Variants), 3; got != want {
		t.Fatalf("PatchChangeKind variant count = %d, want %d", got, want)
	}

	threadItem := unionByName["ThreadItem"]
	if threadItem.Discriminator != "type" {
		t.Fatalf("ThreadItem discriminator = %q, want type", threadItem.Discriminator)
	}
	if got, want := len(threadItem.Variants), 18; got != want {
		t.Fatalf("ThreadItem variant count = %d, want %d", got, want)
	}
	var mcpToolCall TaggedUnionVariantPlan
	var dynamicToolCall TaggedUnionVariantPlan
	var fileChangeItem TaggedUnionVariantPlan
	for _, variant := range threadItem.Variants {
		switch variant.DiscriminatorValue {
		case "mcpToolCall":
			mcpToolCall = variant
		case "dynamicToolCall":
			dynamicToolCall = variant
		case "fileChange":
			fileChangeItem = variant
		}
	}
	mcpToolCallFields := map[string]FieldPlan{}
	for _, field := range mcpToolCall.Fields {
		mcpToolCallFields[field.FieldName] = field
	}
	if mcpToolCallFields["arguments"].GoType != "protocolv2.JSONValue" || mcpToolCallFields["arguments"].Kind != FieldPlanJSONValue {
		t.Fatalf("ThreadItem.mcpToolCall arguments field = %#v", mcpToolCallFields["arguments"])
	}
	if mcpToolCallFields["result"].GoType != "*protocolv2.Nullable[McpToolCallResult]" {
		t.Fatalf("ThreadItem.mcpToolCall result field = %#v", mcpToolCallFields["result"])
	}
	dynamicToolCallFields := map[string]FieldPlan{}
	for _, field := range dynamicToolCall.Fields {
		dynamicToolCallFields[field.FieldName] = field
	}
	if dynamicToolCallFields["arguments"].GoType != "protocolv2.JSONValue" || dynamicToolCallFields["arguments"].Kind != FieldPlanJSONValue {
		t.Fatalf("ThreadItem.dynamicToolCall arguments field = %#v", dynamicToolCallFields["arguments"])
	}
	fileChangeFields := map[string]FieldPlan{}
	for _, field := range fileChangeItem.Fields {
		fileChangeFields[field.FieldName] = field
	}
	if fileChangeFields["changes"].GoType != "[]FileUpdateChange" {
		t.Fatalf("ThreadItem.fileChange changes field = %#v", fileChangeFields["changes"])
	}

	threadStatus := unionByName["ThreadStatus"]
	if threadStatus.Discriminator != "type" {
		t.Fatalf("ThreadStatus discriminator = %q, want type", threadStatus.Discriminator)
	}
	if got, want := len(threadStatus.Variants), 4; got != want {
		t.Fatalf("ThreadStatus variant count = %d, want %d", got, want)
	}
	var activeThreadStatus TaggedUnionVariantPlan
	for _, variant := range threadStatus.Variants {
		switch variant.DiscriminatorValue {
		case "notLoaded", "idle", "systemError":
			if len(variant.Fields) != 0 {
				t.Fatalf("ThreadStatus scalar object variant = %#v", variant)
			}
		case "active":
			activeThreadStatus = variant
		default:
			t.Fatalf("unexpected ThreadStatus variant = %#v", variant)
		}
	}
	activeThreadStatusFields := map[string]FieldPlan{}
	for _, field := range activeThreadStatus.Fields {
		activeThreadStatusFields[field.FieldName] = field
	}
	if activeThreadStatusFields["activeFlags"].GoType != "[]ThreadActiveFlag" ||
		!activeThreadStatusFields["activeFlags"].Required {
		t.Fatalf("ThreadStatus.active fields = %#v", activeThreadStatusFields)
	}

	pluginSource := unionByName["PluginSource"]
	if pluginSource.Discriminator != "type" {
		t.Fatalf("PluginSource discriminator = %q, want type", pluginSource.Discriminator)
	}
	if got, want := len(pluginSource.Variants), 3; got != want {
		t.Fatalf("PluginSource variant count = %d, want %d", got, want)
	}
	var localPluginSource TaggedUnionVariantPlan
	var gitPluginSource TaggedUnionVariantPlan
	var remotePluginSource TaggedUnionVariantPlan
	for _, variant := range pluginSource.Variants {
		switch variant.DiscriminatorValue {
		case "local":
			localPluginSource = variant
		case "git":
			gitPluginSource = variant
		case "remote":
			remotePluginSource = variant
		default:
			t.Fatalf("unexpected PluginSource variant = %#v", variant)
		}
	}
	localPluginSourceFields := map[string]FieldPlan{}
	for _, field := range localPluginSource.Fields {
		localPluginSourceFields[field.FieldName] = field
	}
	if localPluginSourceFields["path"].GoType != "string" || !localPluginSourceFields["path"].Required {
		t.Fatalf("PluginSource.local fields = %#v", localPluginSourceFields)
	}
	gitPluginSourceFields := map[string]FieldPlan{}
	for _, field := range gitPluginSource.Fields {
		gitPluginSourceFields[field.FieldName] = field
	}
	if gitPluginSourceFields["url"].GoType != "string" || !gitPluginSourceFields["url"].Required ||
		gitPluginSourceFields["path"].GoType != "*protocolv2.Nullable[string]" ||
		gitPluginSourceFields["refName"].GoType != "*protocolv2.Nullable[string]" ||
		gitPluginSourceFields["sha"].GoType != "*protocolv2.Nullable[string]" {
		t.Fatalf("PluginSource.git fields = %#v", gitPluginSourceFields)
	}
	if len(remotePluginSource.Fields) != 0 {
		t.Fatalf("PluginSource.remote fields = %#v, want empty", remotePluginSource.Fields)
	}

	reviewTarget := unionByName["ReviewTarget"]
	if reviewTarget.Discriminator != "type" {
		t.Fatalf("ReviewTarget discriminator = %q, want type", reviewTarget.Discriminator)
	}
	if got, want := len(reviewTarget.Variants), 4; got != want {
		t.Fatalf("ReviewTarget variant count = %d, want %d", got, want)
	}
	var baseBranchTarget TaggedUnionVariantPlan
	var commitTarget TaggedUnionVariantPlan
	var customTarget TaggedUnionVariantPlan
	for _, variant := range reviewTarget.Variants {
		switch variant.DiscriminatorValue {
		case "uncommittedChanges":
			if len(variant.Fields) != 0 {
				t.Fatalf("ReviewTarget.uncommittedChanges fields = %#v, want empty", variant.Fields)
			}
		case "baseBranch":
			baseBranchTarget = variant
		case "commit":
			commitTarget = variant
		case "custom":
			customTarget = variant
		default:
			t.Fatalf("unexpected ReviewTarget variant = %#v", variant)
		}
	}
	baseBranchTargetFields := map[string]FieldPlan{}
	for _, field := range baseBranchTarget.Fields {
		baseBranchTargetFields[field.FieldName] = field
	}
	if baseBranchTargetFields["branch"].GoType != "string" || !baseBranchTargetFields["branch"].Required {
		t.Fatalf("ReviewTarget.baseBranch fields = %#v", baseBranchTargetFields)
	}
	commitTargetFields := map[string]FieldPlan{}
	for _, field := range commitTarget.Fields {
		commitTargetFields[field.FieldName] = field
	}
	if commitTargetFields["sha"].GoType != "string" || !commitTargetFields["sha"].Required ||
		commitTargetFields["title"].GoType != "*protocolv2.Nullable[string]" {
		t.Fatalf("ReviewTarget.commit fields = %#v", commitTargetFields)
	}
	customTargetFields := map[string]FieldPlan{}
	for _, field := range customTarget.Fields {
		customTargetFields[field.FieldName] = field
	}
	if customTargetFields["instructions"].GoType != "string" || !customTargetFields["instructions"].Required {
		t.Fatalf("ReviewTarget.custom fields = %#v", customTargetFields)
	}

	threadWebSearchAction := unionByName["WebSearchAction"]
	if threadWebSearchAction.Discriminator != "type" {
		t.Fatalf("WebSearchAction discriminator = %q, want type", threadWebSearchAction.Discriminator)
	}
	if got, want := len(threadWebSearchAction.Variants), 4; got != want {
		t.Fatalf("WebSearchAction variant count = %d, want %d", got, want)
	}
}

func TestGenerateProtocolTypesEmitsNullableDecoder(t *testing.T) {
	generated, err := GenerateProtocolTypes(ProtocolTypePlan{Types: []TypePlan{{
		Kind:       TypePlanObjectStructCandidate,
		SchemaPath: "Example.json",
		TypeName:   "Example",
		Fields: []FieldPlan{{
			FieldName:       "serviceTier",
			GoType:          "*protocolv2.Nullable[string]",
			Kind:            FieldPlanNullableServiceTier,
			Path:            "Example.json#/properties/serviceTier",
			Required:        false,
			WireAllowsNull:  true,
			WireOmitAllowed: true,
		}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, want := range []string{
		"ServiceTier *Nullable[string] `json:\"serviceTier,omitempty\"`",
		`decodeNullableJSONField[string](fields, "serviceTier", "Example.serviceTier", &decoded.ServiceTier)`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated nullable protocol type does not contain %q:\n%s", want, text)
		}
	}
}

func TestGenerateProtocolTypesEmitsRequiredCollectionMarshalGuards(t *testing.T) {
	minItems := uint64(1)
	generated, err := GenerateProtocolTypes(ProtocolTypePlan{Types: []TypePlan{{
		Kind:       TypePlanObjectStructCandidate,
		SchemaPath: "Example.json",
		TypeName:   "Example",
		Fields: []FieldPlan{{
			FieldName:      "items",
			GoType:         "[]string",
			Kind:           FieldPlanArrayString,
			MinItems:       &minItems,
			Path:           "Example.json#/properties/items",
			Required:       true,
			WireAllowsNull: false,
		}, {
			FieldName:      "labels",
			GoType:         "map[string]string",
			Kind:           FieldPlanTypedMap,
			Path:           "Example.json#/properties/labels",
			Required:       true,
			WireAllowsNull: false,
		}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, want := range []string{
		"func (value Example) MarshalJSON() ([]byte, error)",
		`return nil, fmt.Errorf("encode Example.items: nil is not allowed")`,
		`return nil, fmt.Errorf("encode Example.items: must contain at least 1 item")`,
		`return nil, fmt.Errorf("encode Example.labels: nil is not allowed")`,
		`return fmt.Errorf("decode Example.items: must contain at least 1 item")`,
		"return json.Marshal(wire(value))",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated required collection marshal guard does not contain %q:\n%s", want, text)
		}
	}
}

func TestGenerateProtocolTypesEmitsTaggedUnionBoundary(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	generated, err := GenerateProtocolTypes(plan)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, want := range []string{
		"type LoginAccountParams struct {\n\tkind",
		"func NewLoginAccountParamsAPIKey(payload LoginAccountParamsAPIKey) LoginAccountParams",
		"func (value LoginAccountParams) AsAPIKey() (LoginAccountParamsAPIKey, bool)",
		"func (value *LoginAccountParams) UnmarshalJSON(data []byte) error",
		`return unknownUnionVariant("LoginAccountParams", "type", variant)`,
		"type ClientNotification struct {\n\tkind",
		"func NewClientNotificationInitialized() ClientNotification",
		"func (value ClientNotification) AsInitialized() (ClientNotificationInitialized, bool)",
		`return unknownUnionVariant("ClientNotification", "method", variant)`,
		"type ClientRequest struct {\n\tkind",
		"func NewClientRequestThreadStart(payload ClientRequestThreadStart) ClientRequest",
		"func (value ClientRequest) AsMemoryReset() (ClientRequestMemoryReset, bool)",
		`return unknownUnionVariant("ClientRequest", "method", variant)`,
		"type ServerNotification struct {\n\tkind",
		"func NewServerNotificationThreadTokenUsageUpdated(payload ServerNotificationThreadTokenUsageUpdated) ServerNotification",
		"func (value ServerNotification) AsThreadRealtimeSDP() (ServerNotificationThreadRealtimeSDP, bool)",
		`return unknownUnionVariant("ServerNotification", "method", variant)`,
		"type ServerRequest struct {\n\tkind",
		"func NewServerRequestItemCommandExecutionRequestApproval(payload ServerRequestItemCommandExecutionRequestApproval) ServerRequest",
		"func (value ServerRequest) AsItemToolCall() (ServerRequestItemToolCall, bool)",
		`return unknownUnionVariant("ServerRequest", "method", variant)`,
		"type FileChange struct {\n\tkind",
		"func NewFileChangeUpdate(payload FileChangeUpdate) FileChange",
		"func (value FileChange) AsUpdate() (FileChangeUpdate, bool)",
		`return unknownUnionVariant("FileChange", "type", variant)`,
		"type ParsedCommand struct {\n\tkind",
		"func NewParsedCommandSearch(payload ParsedCommandSearch) ParsedCommand",
		"func (value ParsedCommand) AsSearch() (ParsedCommandSearch, bool)",
		`return unknownUnionVariant("ParsedCommand", "type", variant)`,
		"type DynamicToolCallOutputContentItem struct {\n\tkind",
		"func NewDynamicToolCallOutputContentItemInputText(payload DynamicToolCallOutputContentItemInputText) DynamicToolCallOutputContentItem",
		"func (value DynamicToolCallOutputContentItem) AsInputImage() (DynamicToolCallOutputContentItemInputImage, bool)",
		`return unknownUnionVariant("DynamicToolCallOutputContentItem", "type", variant)`,
		"type Account struct {\n\tkind",
		"func NewAccountChatGPT(payload AccountChatGPT) Account",
		"func (value Account) AsAmazonBedrock() (AccountAmazonBedrock, bool)",
		`return unknownUnionVariant("Account", "type", variant)`,
		"type SandboxPolicy struct {\n\tkind",
		"func NewSandboxPolicyWorkspaceWrite(payload SandboxPolicyWorkspaceWrite) SandboxPolicy",
		"func (value SandboxPolicy) AsReadOnly() (SandboxPolicyReadOnly, bool)",
		`return unknownUnionVariant("SandboxPolicy", "type", variant)`,
		"type ConfigLayerSource struct {\n\tkind",
		"func NewConfigLayerSourceMdm(payload ConfigLayerSourceMdm) ConfigLayerSource",
		"func (value ConfigLayerSource) AsSessionFlags() (ConfigLayerSourceSessionFlags, bool)",
		`return unknownUnionVariant("ConfigLayerSource", "type", variant)`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated tagged union output does not contain %q", want)
		}
	}
	for _, forbidden := range []string{
		"APIKey *LoginAccountParamsAPIKey",
		"UnknownVariant",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("generated tagged union output contains forbidden marker %q", forbidden)
		}
	}
}

func TestGenerateProtocolTypesEmitsScalarUnionBoundary(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	generated, err := GenerateProtocolTypes(plan)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, want := range []string{
		"type RequestId struct {\n\tkind",
		"func NewRequestIdString(value string) RequestId",
		"func NewRequestIdInt64(value int64) RequestId",
		"func (value RequestId) AsString() (string, bool)",
		"func (value RequestId) AsInt64() (int64, bool)",
		"func (value *RequestId) UnmarshalJSON(data []byte) error",
		"func NewThreadListCwdFilterArray(value []string) ThreadListCwdFilter",
		`return fmt.Errorf("decode ThreadListCwdFilter: expected array item %d to be string", index)`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated scalar union output does not contain %q", want)
		}
	}
	for _, forbidden := range []string{
		"StringValue",
		"Int64Value",
		"UnknownVariant",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("generated scalar union output contains forbidden marker %q", forbidden)
		}
	}
}

func TestGenerateProtocolTypesDoesNotExposeJSONRPCEnvelopeSurface(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	generated, err := GenerateProtocolTypes(plan)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, forbidden := range []string{
		"type JSONRPCError ",
		"type JSONRPCNotification ",
		"type JSONRPCRequest ",
		"type JSONRPCResponse ",
		"type JSONRPCMessage ",
		"func NewJSONRPCMessage",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("generated protocolv2 output exposes JSON-RPC envelope surface %q", forbidden)
		}
	}
}

func TestGenerateProtocolTypesEmitsMixedUnionBoundary(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	generated, err := GenerateProtocolTypes(plan)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, want := range []string{
		"type ReviewDecision struct {\n\tkind",
		"type CommandExecutionApprovalDecision struct {\n\tkind",
		"func NewCommandExecutionApprovalDecisionAcceptWithExecpolicyAmendment(payload CommandExecutionApprovalDecisionAcceptWithExecpolicyAmendment) CommandExecutionApprovalDecision",
		"func NewCommandExecutionApprovalDecisionApplyNetworkPolicyAmendment(payload CommandExecutionApprovalDecisionApplyNetworkPolicyAmendment) CommandExecutionApprovalDecision",
		"func (value CommandExecutionApprovalDecision) AsApplyNetworkPolicyAmendment() (CommandExecutionApprovalDecisionApplyNetworkPolicyAmendment, bool)",
		`return unknownUnionVariant("CommandExecutionApprovalDecision", "value", variant)`,
		"func NewReviewDecisionApproved() ReviewDecision",
		"func NewReviewDecisionApprovedExecpolicyAmendment(payload ReviewDecisionApprovedExecpolicyAmendment) ReviewDecision",
		"func NewReviewDecisionNetworkPolicyAmendment(payload ReviewDecisionNetworkPolicyAmendment) ReviewDecision",
		"func (value ReviewDecision) AsNetworkPolicyAmendment() (ReviewDecisionNetworkPolicyAmendment, bool)",
		`return unknownUnionVariant("ReviewDecision", "value", variant)`,
		`return nil, fmt.Errorf("encode ReviewDecision.approved_execpolicy_amendment.proposed_execpolicy_amendment: nil is not allowed")`,
		"type NetworkPolicyAmendment struct {",
		"Decision ReviewDecision `json:\"decision\"`",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated mixed union output does not contain %q", want)
		}
	}
	for _, forbidden := range []string{
		"ReviewDecision string",
		"ReviewDecisionUnknownVariant",
		"CommandExecutionApprovalDecision string",
		"CommandExecutionApprovalDecisionUnknownVariant",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("generated mixed union output contains forbidden marker %q", forbidden)
		}
	}
}

func TestGeneratedProtocolTypesKeepTypedBoundary(t *testing.T) {
	plan, err := BuildProtocolTypePlan(filepath.Join("..", "protocolschema", "appserver", "v2"))
	if err != nil {
		t.Fatal(err)
	}
	generated, err := GenerateProtocolTypes(plan)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, forbidden := range []string{"json.RawMessage", "map[string]any", "interface{}", "UnknownFields", "AdditionalFields"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("generated protocol types contain forbidden public passthrough marker %q", forbidden)
		}
	}
}

func TestFirstPassSelectionRejectsRefMapLeafTypes(t *testing.T) {
	typ := TypePlan{
		Kind:       TypePlanObjectStructCandidate,
		SchemaPath: "Example.json",
		TypeName:   "Example",
		Fields: []FieldPlan{{
			FieldName: "answers",
			GoType:    "map[string]ToolRequestUserInputAnswer",
			Kind:      FieldPlanTypedMap,
			Path:      "Example.json#/properties/answers",
			Required:  true,
		}},
	}
	selected, err := SelectFirstPassGeneratedTypes(ProtocolTypePlan{Types: []TypePlan{typ}})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 0 {
		t.Fatalf("ref map leaf type was selected: %#v", selected)
	}
}

func TestGeneratedTypeSelectionRejectsEnumStructNameCollision(t *testing.T) {
	enumSchema := &Schema{
		Type: SchemaTypeSet{Values: []string{"object"}},
		Definitions: map[string]*Schema{
			"Example": {
				Type: SchemaTypeSet{Values: []string{"string"}},
				Enum: []string{"known"},
			},
		},
	}
	typ := TypePlan{
		Kind:       TypePlanEmptyStructCandidate,
		Schema:     enumSchema,
		SchemaPath: "Example.json",
		TypeName:   "Example",
	}
	_, err := SelectFirstPassGeneratedTypes(ProtocolTypePlan{Types: []TypePlan{typ}})
	if err == nil {
		t.Fatal("expected generated enum/struct name collision to fail")
	}
	if !strings.Contains(err.Error(), "conflicts with generated enum type") {
		t.Fatalf("unexpected collision error: %v", err)
	}
}

func TestFieldGoNameUsesGoAcronyms(t *testing.T) {
	cases := map[string]string{
		"authorizationUrl": "AuthorizationURL",
		"chatgptAccountId": "ChatGPTAccountID",
		"cwds":             "CWDs",
		"httpStatusCode":   "HTTPStatusCode",
		"threadIds":        "ThreadIDs",
		"threadId":         "ThreadID",
		"uri":              "URI",
	}
	for field, want := range cases {
		if got := fieldGoName(field); got != want {
			t.Fatalf("fieldGoName(%q) = %q, want %q", field, got, want)
		}
	}
}

func TestLeafGoTypePeelsNullableArrays(t *testing.T) {
	if got, want := leafGoType("*Nullable[[]ToolRequestUserInputOption]"), "ToolRequestUserInputOption"; got != want {
		t.Fatalf("leafGoType nullable array = %q, want %q", got, want)
	}
}

func TestGeneratedDefinitionStructCheckpointsCoverRust0142Drift(t *testing.T) {
	cases := []struct {
		schemaPath string
		name       string
	}{
		{"v2/ExternalAgentConfigImportHistoriesReadResponse.json", "ExternalAgentConfigImportHistory"},
		{"v2/GetWorkspaceMessagesResponse.json", "WorkspaceMessage"},
		{"v2/ThreadResumeParams.json", "InternalChatMessageMetadataPassthrough"},
		{"v2/TurnStartResponse.json", "McpToolCallAppContext"},
	}
	for _, tc := range cases {
		if !isGeneratedDefinitionStructCheckpoint(tc.schemaPath, tc.name) {
			t.Fatalf("%s %s was not selected as a generated definition struct checkpoint", tc.schemaPath, tc.name)
		}
	}
}

func TestGeneratedDefinitionEnumCheckpointsCoverRust0142Drift(t *testing.T) {
	if !isGeneratedDefinitionStringEnumCheckpoint("v2/GetWorkspaceMessagesResponse.json", "WorkspaceMessageType") {
		t.Fatal("WorkspaceMessageType was not selected as a generated definition enum checkpoint")
	}
}
