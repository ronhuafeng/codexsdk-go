package codexsdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ronhuafeng/codexsdk-go/codexsdk/protocolv2"
)

type appsFacade struct {
	client *client
}

type accountsFacade struct {
	client *client
}

type commandsFacade struct {
	client *client
}

type collaborationModesFacade struct {
	client *client
}

type configFacade struct {
	client *client
}

type configRequirementsFacade struct {
	client *client
}

type experimentalFeaturesFacade struct {
	client *client
}

type externalAgentConfigsFacade struct {
	client *client
}

type feedbackFacade struct {
	client *client
}

type fsFacade struct {
	client *client
}

type fuzzyFileSearchFacade struct {
	client *client
}

type hooksFacade struct {
	client *client
}

type marketplaceFacade struct {
	client *client
}

type memoryFacade struct {
	client *client
}

type mockFacade struct {
	client *client
}

type pluginsFacade struct {
	client *client
}

type processesFacade struct {
	client *client
}

type reviewsFacade struct {
	client *client
}

type mcpServersFacade struct {
	client *client
}

type mcpServerStatusFacade struct {
	client *client
}

type modelProvidersFacade struct {
	client *client
}

type modelsFacade struct {
	client *client
}

type skillsFacade struct {
	client *client
}

type threadsFacade struct {
	client *client
}

type turnsFacade struct {
	client *client
}

type windowsSandboxFacade struct {
	client *client
}

func (c *client) Apps() Apps {
	return appsFacade{client: c}
}

func (c *client) Accounts() Accounts {
	return accountsFacade{client: c}
}

func (c *client) Commands() Commands {
	return commandsFacade{client: c}
}

func (c *client) CollaborationModes() CollaborationModes {
	return collaborationModesFacade{client: c}
}

func (c *client) Config() Config {
	return configFacade{client: c}
}

func (c *client) ConfigRequirements() ConfigRequirements {
	return configRequirementsFacade{client: c}
}

func (c *client) ExperimentalFeatures() ExperimentalFeatures {
	return experimentalFeaturesFacade{client: c}
}

func (c *client) ExternalAgentConfigs() ExternalAgentConfigs {
	return externalAgentConfigsFacade{client: c}
}

func (c *client) Feedback() Feedback {
	return feedbackFacade{client: c}
}

func (c *client) FS() FS {
	return fsFacade{client: c}
}

func (c *client) FuzzyFileSearch() FuzzyFileSearch {
	return fuzzyFileSearchFacade{client: c}
}

func (c *client) Hooks() Hooks {
	return hooksFacade{client: c}
}

func (c *client) Marketplace() Marketplace {
	return marketplaceFacade{client: c}
}

func (c *client) Memory() Memory {
	return memoryFacade{client: c}
}

func (c *client) Mock() Mock {
	return mockFacade{client: c}
}

func (c *client) Plugins() Plugins {
	return pluginsFacade{client: c}
}

func (c *client) Processes() Processes {
	return processesFacade{client: c}
}

func (c *client) Reviews() Reviews {
	return reviewsFacade{client: c}
}

func (c *client) MCPServers() MCPServers {
	return mcpServersFacade{client: c}
}

func (c *client) MCPServerStatus() MCPServerStatus {
	return mcpServerStatusFacade{client: c}
}

func (c *client) ModelProviders() ModelProviders {
	return modelProvidersFacade{client: c}
}

func (c *client) Models() Models {
	return modelsFacade{client: c}
}

func (c *client) Skills() Skills {
	return skillsFacade{client: c}
}

func (c *client) Threads() Threads {
	return threadsFacade{client: c}
}

func (c *client) Turns() Turns {
	return turnsFacade{client: c}
}

func (c *client) WindowsSandbox() WindowsSandbox {
	return windowsSandboxFacade{client: c}
}

func (f appsFacade) List(ctx context.Context, params protocolv2.AppsListParams) (protocolv2.AppsListResponse, error) {
	var response protocolv2.AppsListResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodAppList, params, &response); err != nil {
		return protocolv2.AppsListResponse{}, err
	}
	return response, nil
}

func (f accountsFacade) LoginCancel(ctx context.Context, params protocolv2.CancelLoginAccountParams) (protocolv2.CancelLoginAccountResponse, error) {
	var response protocolv2.CancelLoginAccountResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodAccountLoginCancel, params, &response); err != nil {
		return protocolv2.CancelLoginAccountResponse{}, err
	}
	return response, nil
}

func (f accountsFacade) LoginStart(ctx context.Context, params protocolv2.LoginAccountParams) (protocolv2.LoginAccountResponse, error) {
	var response protocolv2.LoginAccountResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodAccountLoginStart, params, &response); err != nil {
		return protocolv2.LoginAccountResponse{}, err
	}
	return response, nil
}

func (f accountsFacade) Logout(ctx context.Context) (protocolv2.LogoutAccountResponse, error) {
	var response protocolv2.LogoutAccountResponse
	if err := f.client.callProtocolNoParams(ctx, protocolv2.MethodAccountLogout, &response); err != nil {
		return protocolv2.LogoutAccountResponse{}, err
	}
	return response, nil
}

func (f accountsFacade) RateLimitsRead(ctx context.Context) (protocolv2.GetAccountRateLimitsResponse, error) {
	var response protocolv2.GetAccountRateLimitsResponse
	if err := f.client.callProtocolNoParams(ctx, protocolv2.MethodAccountRateLimitsRead, &response); err != nil {
		return protocolv2.GetAccountRateLimitsResponse{}, err
	}
	return response, nil
}

func (f accountsFacade) Read(ctx context.Context, params protocolv2.GetAccountParams) (protocolv2.GetAccountResponse, error) {
	var response protocolv2.GetAccountResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodAccountRead, params, &response); err != nil {
		return protocolv2.GetAccountResponse{}, err
	}
	return response, nil
}

func (f accountsFacade) SendAddCreditsNudgeEmail(ctx context.Context, params protocolv2.SendAddCreditsNudgeEmailParams) (protocolv2.SendAddCreditsNudgeEmailResponse, error) {
	var response protocolv2.SendAddCreditsNudgeEmailResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodAccountSendAddCreditsNudgeEmail, params, &response); err != nil {
		return protocolv2.SendAddCreditsNudgeEmailResponse{}, err
	}
	return response, nil
}

func (f commandsFacade) Exec(ctx context.Context, params protocolv2.CommandExecParams) (protocolv2.CommandExecResponse, error) {
	var response protocolv2.CommandExecResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodCommandExec, params, &response); err != nil {
		return protocolv2.CommandExecResponse{}, err
	}
	return response, nil
}

func (f commandsFacade) ExecResize(ctx context.Context, params protocolv2.CommandExecResizeParams) (protocolv2.CommandExecResizeResponse, error) {
	var response protocolv2.CommandExecResizeResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodCommandExecResize, params, &response); err != nil {
		return protocolv2.CommandExecResizeResponse{}, err
	}
	return response, nil
}

func (f commandsFacade) ExecTerminate(ctx context.Context, params protocolv2.CommandExecTerminateParams) (protocolv2.CommandExecTerminateResponse, error) {
	var response protocolv2.CommandExecTerminateResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodCommandExecTerminate, params, &response); err != nil {
		return protocolv2.CommandExecTerminateResponse{}, err
	}
	return response, nil
}

func (f commandsFacade) ExecWrite(ctx context.Context, params protocolv2.CommandExecWriteParams) (protocolv2.CommandExecWriteResponse, error) {
	var response protocolv2.CommandExecWriteResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodCommandExecWrite, params, &response); err != nil {
		return protocolv2.CommandExecWriteResponse{}, err
	}
	return response, nil
}

func (f collaborationModesFacade) List(ctx context.Context, params protocolv2.CollaborationModeListParams) (protocolv2.CollaborationModeListResponse, error) {
	var response protocolv2.CollaborationModeListResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodCollaborationModeList, params, &response); err != nil {
		return protocolv2.CollaborationModeListResponse{}, err
	}
	return response, nil
}

func (f configFacade) BatchWrite(ctx context.Context, params protocolv2.ConfigBatchWriteParams) (protocolv2.ConfigWriteResponse, error) {
	var response protocolv2.ConfigWriteResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodConfigBatchWrite, params, &response); err != nil {
		return protocolv2.ConfigWriteResponse{}, err
	}
	return response, nil
}

func (f configFacade) MCPServerReload(ctx context.Context) (protocolv2.McpServerRefreshResponse, error) {
	var response protocolv2.McpServerRefreshResponse
	if err := f.client.callProtocolNoParams(ctx, protocolv2.MethodConfigMCPServerReload, &response); err != nil {
		return protocolv2.McpServerRefreshResponse{}, err
	}
	return response, nil
}

func (f configFacade) Read(ctx context.Context, params protocolv2.ConfigReadParams) (protocolv2.ConfigReadResponse, error) {
	var response protocolv2.ConfigReadResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodConfigRead, params, &response); err != nil {
		return protocolv2.ConfigReadResponse{}, err
	}
	return response, nil
}

func (f configFacade) ValueWrite(ctx context.Context, params protocolv2.ConfigValueWriteParams) (protocolv2.ConfigWriteResponse, error) {
	var response protocolv2.ConfigWriteResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodConfigValueWrite, params, &response); err != nil {
		return protocolv2.ConfigWriteResponse{}, err
	}
	return response, nil
}

func (f configRequirementsFacade) Read(ctx context.Context) (protocolv2.ConfigRequirementsReadResponse, error) {
	var response protocolv2.ConfigRequirementsReadResponse
	if err := f.client.callProtocolNoParams(ctx, protocolv2.MethodConfigRequirementsRead, &response); err != nil {
		return protocolv2.ConfigRequirementsReadResponse{}, err
	}
	return response, nil
}

func (f experimentalFeaturesFacade) EnablementSet(ctx context.Context, params protocolv2.ExperimentalFeatureEnablementSetParams) (protocolv2.ExperimentalFeatureEnablementSetResponse, error) {
	var response protocolv2.ExperimentalFeatureEnablementSetResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodExperimentalFeatureEnablementSet, params, &response); err != nil {
		return protocolv2.ExperimentalFeatureEnablementSetResponse{}, err
	}
	return response, nil
}

func (f experimentalFeaturesFacade) List(ctx context.Context, params protocolv2.ExperimentalFeatureListParams) (protocolv2.ExperimentalFeatureListResponse, error) {
	var response protocolv2.ExperimentalFeatureListResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodExperimentalFeatureList, params, &response); err != nil {
		return protocolv2.ExperimentalFeatureListResponse{}, err
	}
	return response, nil
}

func (f externalAgentConfigsFacade) Detect(ctx context.Context, params protocolv2.ExternalAgentConfigDetectParams) (protocolv2.ExternalAgentConfigDetectResponse, error) {
	var response protocolv2.ExternalAgentConfigDetectResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodExternalAgentConfigDetect, params, &response); err != nil {
		return protocolv2.ExternalAgentConfigDetectResponse{}, err
	}
	return response, nil
}

func (f externalAgentConfigsFacade) Import(ctx context.Context, params protocolv2.ExternalAgentConfigImportParams) (protocolv2.ExternalAgentConfigImportResponse, error) {
	var response protocolv2.ExternalAgentConfigImportResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodExternalAgentConfigImport, params, &response); err != nil {
		return protocolv2.ExternalAgentConfigImportResponse{}, err
	}
	return response, nil
}

func (f feedbackFacade) Upload(ctx context.Context, params protocolv2.FeedbackUploadParams) (protocolv2.FeedbackUploadResponse, error) {
	var response protocolv2.FeedbackUploadResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodFeedbackUpload, params, &response); err != nil {
		return protocolv2.FeedbackUploadResponse{}, err
	}
	return response, nil
}

func (f fsFacade) Copy(ctx context.Context, params protocolv2.FsCopyParams) (protocolv2.FsCopyResponse, error) {
	var response protocolv2.FsCopyResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodFSCopy, params, &response); err != nil {
		return protocolv2.FsCopyResponse{}, err
	}
	return response, nil
}

func (f fsFacade) CreateDirectory(ctx context.Context, params protocolv2.FsCreateDirectoryParams) (protocolv2.FsCreateDirectoryResponse, error) {
	var response protocolv2.FsCreateDirectoryResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodFSCreateDirectory, params, &response); err != nil {
		return protocolv2.FsCreateDirectoryResponse{}, err
	}
	return response, nil
}

func (f fsFacade) GetMetadata(ctx context.Context, params protocolv2.FsGetMetadataParams) (protocolv2.FsGetMetadataResponse, error) {
	var response protocolv2.FsGetMetadataResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodFSGetMetadata, params, &response); err != nil {
		return protocolv2.FsGetMetadataResponse{}, err
	}
	return response, nil
}

func (f fsFacade) ReadDirectory(ctx context.Context, params protocolv2.FsReadDirectoryParams) (protocolv2.FsReadDirectoryResponse, error) {
	var response protocolv2.FsReadDirectoryResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodFSReadDirectory, params, &response); err != nil {
		return protocolv2.FsReadDirectoryResponse{}, err
	}
	return response, nil
}

func (f fsFacade) ReadFile(ctx context.Context, params protocolv2.FsReadFileParams) (protocolv2.FsReadFileResponse, error) {
	var response protocolv2.FsReadFileResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodFSReadFile, params, &response); err != nil {
		return protocolv2.FsReadFileResponse{}, err
	}
	return response, nil
}

func (f fsFacade) Remove(ctx context.Context, params protocolv2.FsRemoveParams) (protocolv2.FsRemoveResponse, error) {
	var response protocolv2.FsRemoveResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodFSRemove, params, &response); err != nil {
		return protocolv2.FsRemoveResponse{}, err
	}
	return response, nil
}

func (f fsFacade) Unwatch(ctx context.Context, params protocolv2.FsUnwatchParams) (protocolv2.FsUnwatchResponse, error) {
	var response protocolv2.FsUnwatchResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodFSUnwatch, params, &response); err != nil {
		return protocolv2.FsUnwatchResponse{}, err
	}
	return response, nil
}

func (f fsFacade) Watch(ctx context.Context, params protocolv2.FsWatchParams) (protocolv2.FsWatchResponse, error) {
	var response protocolv2.FsWatchResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodFSWatch, params, &response); err != nil {
		return protocolv2.FsWatchResponse{}, err
	}
	return response, nil
}

func (f fsFacade) WriteFile(ctx context.Context, params protocolv2.FsWriteFileParams) (protocolv2.FsWriteFileResponse, error) {
	var response protocolv2.FsWriteFileResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodFSWriteFile, params, &response); err != nil {
		return protocolv2.FsWriteFileResponse{}, err
	}
	return response, nil
}

func (f fuzzyFileSearchFacade) Search(ctx context.Context, params protocolv2.FuzzyFileSearchParams) (protocolv2.FuzzyFileSearchResponse, error) {
	var response protocolv2.FuzzyFileSearchResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodFuzzyFileSearch, params, &response); err != nil {
		return protocolv2.FuzzyFileSearchResponse{}, err
	}
	return response, nil
}

func (f fuzzyFileSearchFacade) SessionStart(ctx context.Context, params protocolv2.FuzzyFileSearchSessionStartParams) (protocolv2.FuzzyFileSearchSessionStartResponse, error) {
	var response protocolv2.FuzzyFileSearchSessionStartResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodFuzzyFileSearchSessionStart, params, &response); err != nil {
		return protocolv2.FuzzyFileSearchSessionStartResponse{}, err
	}
	return response, nil
}

func (f fuzzyFileSearchFacade) SessionStop(ctx context.Context, params protocolv2.FuzzyFileSearchSessionStopParams) (protocolv2.FuzzyFileSearchSessionStopResponse, error) {
	var response protocolv2.FuzzyFileSearchSessionStopResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodFuzzyFileSearchSessionStop, params, &response); err != nil {
		return protocolv2.FuzzyFileSearchSessionStopResponse{}, err
	}
	return response, nil
}

func (f fuzzyFileSearchFacade) SessionUpdate(ctx context.Context, params protocolv2.FuzzyFileSearchSessionUpdateParams) (protocolv2.FuzzyFileSearchSessionUpdateResponse, error) {
	var response protocolv2.FuzzyFileSearchSessionUpdateResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodFuzzyFileSearchSessionUpdate, params, &response); err != nil {
		return protocolv2.FuzzyFileSearchSessionUpdateResponse{}, err
	}
	return response, nil
}

func (f hooksFacade) List(ctx context.Context, params protocolv2.HooksListParams) (protocolv2.HooksListResponse, error) {
	var response protocolv2.HooksListResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodHooksList, params, &response); err != nil {
		return protocolv2.HooksListResponse{}, err
	}
	return response, nil
}

func (f marketplaceFacade) Add(ctx context.Context, params protocolv2.MarketplaceAddParams) (protocolv2.MarketplaceAddResponse, error) {
	var response protocolv2.MarketplaceAddResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodMarketplaceAdd, params, &response); err != nil {
		return protocolv2.MarketplaceAddResponse{}, err
	}
	return response, nil
}

func (f marketplaceFacade) Remove(ctx context.Context, params protocolv2.MarketplaceRemoveParams) (protocolv2.MarketplaceRemoveResponse, error) {
	var response protocolv2.MarketplaceRemoveResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodMarketplaceRemove, params, &response); err != nil {
		return protocolv2.MarketplaceRemoveResponse{}, err
	}
	return response, nil
}

func (f marketplaceFacade) Upgrade(ctx context.Context, params protocolv2.MarketplaceUpgradeParams) (protocolv2.MarketplaceUpgradeResponse, error) {
	var response protocolv2.MarketplaceUpgradeResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodMarketplaceUpgrade, params, &response); err != nil {
		return protocolv2.MarketplaceUpgradeResponse{}, err
	}
	return response, nil
}

func (f memoryFacade) Reset(ctx context.Context) (protocolv2.MemoryResetResponse, error) {
	var response protocolv2.MemoryResetResponse
	if err := f.client.callProtocolNoParams(ctx, protocolv2.MethodMemoryReset, &response); err != nil {
		return protocolv2.MemoryResetResponse{}, err
	}
	return response, nil
}

func (f mockFacade) ExperimentalMethod(ctx context.Context, params protocolv2.MockExperimentalMethodParams) (protocolv2.MockExperimentalMethodResponse, error) {
	var response protocolv2.MockExperimentalMethodResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodMockExperimentalMethod, params, &response); err != nil {
		return protocolv2.MockExperimentalMethodResponse{}, err
	}
	return response, nil
}

func (f pluginsFacade) Install(ctx context.Context, params protocolv2.PluginInstallParams) (protocolv2.PluginInstallResponse, error) {
	var response protocolv2.PluginInstallResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodPluginInstall, params, &response); err != nil {
		return protocolv2.PluginInstallResponse{}, err
	}
	return response, nil
}

func (f pluginsFacade) List(ctx context.Context, params protocolv2.PluginListParams) (protocolv2.PluginListResponse, error) {
	var response protocolv2.PluginListResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodPluginList, params, &response); err != nil {
		return protocolv2.PluginListResponse{}, err
	}
	return response, nil
}

func (f pluginsFacade) Read(ctx context.Context, params protocolv2.PluginReadParams) (protocolv2.PluginReadResponse, error) {
	var response protocolv2.PluginReadResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodPluginRead, params, &response); err != nil {
		return protocolv2.PluginReadResponse{}, err
	}
	return response, nil
}

func (f pluginsFacade) ShareDelete(ctx context.Context, params protocolv2.PluginShareDeleteParams) (protocolv2.PluginShareDeleteResponse, error) {
	var response protocolv2.PluginShareDeleteResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodPluginShareDelete, params, &response); err != nil {
		return protocolv2.PluginShareDeleteResponse{}, err
	}
	return response, nil
}

func (f pluginsFacade) ShareList(ctx context.Context, params protocolv2.PluginShareListParams) (protocolv2.PluginShareListResponse, error) {
	var response protocolv2.PluginShareListResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodPluginShareList, params, &response); err != nil {
		return protocolv2.PluginShareListResponse{}, err
	}
	return response, nil
}

func (f pluginsFacade) ShareSave(ctx context.Context, params protocolv2.PluginShareSaveParams) (protocolv2.PluginShareSaveResponse, error) {
	var response protocolv2.PluginShareSaveResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodPluginShareSave, params, &response); err != nil {
		return protocolv2.PluginShareSaveResponse{}, err
	}
	return response, nil
}

func (f pluginsFacade) ShareUpdateTargets(ctx context.Context, params protocolv2.PluginShareUpdateTargetsParams) (protocolv2.PluginShareUpdateTargetsResponse, error) {
	var response protocolv2.PluginShareUpdateTargetsResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodPluginShareUpdateTargets, params, &response); err != nil {
		return protocolv2.PluginShareUpdateTargetsResponse{}, err
	}
	return response, nil
}

func (f pluginsFacade) SkillRead(ctx context.Context, params protocolv2.PluginSkillReadParams) (protocolv2.PluginSkillReadResponse, error) {
	var response protocolv2.PluginSkillReadResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodPluginSkillRead, params, &response); err != nil {
		return protocolv2.PluginSkillReadResponse{}, err
	}
	return response, nil
}

func (f pluginsFacade) Uninstall(ctx context.Context, params protocolv2.PluginUninstallParams) (protocolv2.PluginUninstallResponse, error) {
	var response protocolv2.PluginUninstallResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodPluginUninstall, params, &response); err != nil {
		return protocolv2.PluginUninstallResponse{}, err
	}
	return response, nil
}

func (f processesFacade) Kill(ctx context.Context, params protocolv2.ProcessKillParams) (protocolv2.ProcessKillResponse, error) {
	var response protocolv2.ProcessKillResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodProcessKill, params, &response); err != nil {
		return protocolv2.ProcessKillResponse{}, err
	}
	return response, nil
}

func (f processesFacade) ResizePTY(ctx context.Context, params protocolv2.ProcessResizePtyParams) (protocolv2.ProcessResizePtyResponse, error) {
	var response protocolv2.ProcessResizePtyResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodProcessResizePTY, params, &response); err != nil {
		return protocolv2.ProcessResizePtyResponse{}, err
	}
	return response, nil
}

func (f processesFacade) Spawn(ctx context.Context, params protocolv2.ProcessSpawnParams) (protocolv2.ProcessSpawnResponse, error) {
	var response protocolv2.ProcessSpawnResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodProcessSpawn, params, &response); err != nil {
		return protocolv2.ProcessSpawnResponse{}, err
	}
	return response, nil
}

func (f processesFacade) WriteStdin(ctx context.Context, params protocolv2.ProcessWriteStdinParams) (protocolv2.ProcessWriteStdinResponse, error) {
	var response protocolv2.ProcessWriteStdinResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodProcessWriteStdin, params, &response); err != nil {
		return protocolv2.ProcessWriteStdinResponse{}, err
	}
	return response, nil
}

func (f mcpServersFacade) OAuthLogin(ctx context.Context, params protocolv2.McpServerOauthLoginParams) (protocolv2.McpServerOauthLoginResponse, error) {
	var response protocolv2.McpServerOauthLoginResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodMCPServerOAuthLogin, params, &response); err != nil {
		return protocolv2.McpServerOauthLoginResponse{}, err
	}
	return response, nil
}

func (f mcpServersFacade) ResourceRead(ctx context.Context, params protocolv2.McpResourceReadParams) (protocolv2.McpResourceReadResponse, error) {
	var response protocolv2.McpResourceReadResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodMCPServerResourceRead, params, &response); err != nil {
		return protocolv2.McpResourceReadResponse{}, err
	}
	return response, nil
}

func (f mcpServersFacade) ToolCall(ctx context.Context, params protocolv2.McpServerToolCallParams) (protocolv2.McpServerToolCallResponse, error) {
	var response protocolv2.McpServerToolCallResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodMCPServerToolCall, params, &response); err != nil {
		return protocolv2.McpServerToolCallResponse{}, err
	}
	return response, nil
}

func (f mcpServerStatusFacade) List(ctx context.Context, params protocolv2.ListMcpServerStatusParams) (protocolv2.ListMcpServerStatusResponse, error) {
	var response protocolv2.ListMcpServerStatusResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodMCPServerStatusList, params, &response); err != nil {
		return protocolv2.ListMcpServerStatusResponse{}, err
	}
	return response, nil
}

func (f modelProvidersFacade) CapabilitiesRead(ctx context.Context, params protocolv2.ModelProviderCapabilitiesReadParams) (protocolv2.ModelProviderCapabilitiesReadResponse, error) {
	var response protocolv2.ModelProviderCapabilitiesReadResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodModelProviderCapabilitiesRead, params, &response); err != nil {
		return protocolv2.ModelProviderCapabilitiesReadResponse{}, err
	}
	return response, nil
}

func (f modelsFacade) List(ctx context.Context, params protocolv2.ModelListParams) (protocolv2.ModelListResponse, error) {
	var response protocolv2.ModelListResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodModelList, params, &response); err != nil {
		return protocolv2.ModelListResponse{}, err
	}
	return response, nil
}

func (f reviewsFacade) Start(ctx context.Context, params protocolv2.ReviewStartParams) (protocolv2.ReviewStartResponse, error) {
	var response protocolv2.ReviewStartResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodReviewStart, params, &response); err != nil {
		return protocolv2.ReviewStartResponse{}, err
	}
	return response, nil
}

func (f skillsFacade) ConfigWrite(ctx context.Context, params protocolv2.SkillsConfigWriteParams) (protocolv2.SkillsConfigWriteResponse, error) {
	var response protocolv2.SkillsConfigWriteResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodSkillsConfigWrite, params, &response); err != nil {
		return protocolv2.SkillsConfigWriteResponse{}, err
	}
	return response, nil
}

func (f skillsFacade) List(ctx context.Context, params protocolv2.SkillsListParams) (protocolv2.SkillsListResponse, error) {
	var response protocolv2.SkillsListResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodSkillsList, params, &response); err != nil {
		return protocolv2.SkillsListResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) ApproveGuardianDeniedAction(ctx context.Context, params protocolv2.ThreadApproveGuardianDeniedActionParams) (protocolv2.ThreadApproveGuardianDeniedActionResponse, error) {
	var response protocolv2.ThreadApproveGuardianDeniedActionResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadApproveGuardianDeniedAction, params, &response); err != nil {
		return protocolv2.ThreadApproveGuardianDeniedActionResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) Archive(ctx context.Context, params protocolv2.ThreadArchiveParams) (protocolv2.ThreadArchiveResponse, error) {
	var response protocolv2.ThreadArchiveResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadArchive, params, &response); err != nil {
		return protocolv2.ThreadArchiveResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) BackgroundTerminalsClean(ctx context.Context, params protocolv2.ThreadBackgroundTerminalsCleanParams) (protocolv2.ThreadBackgroundTerminalsCleanResponse, error) {
	var response protocolv2.ThreadBackgroundTerminalsCleanResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadBackgroundTerminalsClean, params, &response); err != nil {
		return protocolv2.ThreadBackgroundTerminalsCleanResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) CompactStart(ctx context.Context, params protocolv2.ThreadCompactStartParams) (protocolv2.ThreadCompactStartResponse, error) {
	var response protocolv2.ThreadCompactStartResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadCompactStart, params, &response); err != nil {
		return protocolv2.ThreadCompactStartResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) DecrementElicitation(ctx context.Context, params protocolv2.ThreadDecrementElicitationParams) (protocolv2.ThreadDecrementElicitationResponse, error) {
	var response protocolv2.ThreadDecrementElicitationResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadDecrementElicitation, params, &response); err != nil {
		return protocolv2.ThreadDecrementElicitationResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) Start(ctx context.Context, params protocolv2.ThreadStartParams) (protocolv2.ThreadStartResponse, error) {
	var response protocolv2.ThreadStartResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadStart, params, &response); err != nil {
		return protocolv2.ThreadStartResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) Resume(ctx context.Context, params protocolv2.ThreadResumeParams) (protocolv2.ThreadResumeResponse, error) {
	var response protocolv2.ThreadResumeResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadResume, params, &response); err != nil {
		return protocolv2.ThreadResumeResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) InjectItems(ctx context.Context, params protocolv2.ThreadInjectItemsParams) (protocolv2.ThreadInjectItemsResponse, error) {
	var response protocolv2.ThreadInjectItemsResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadInjectItems, params, &response); err != nil {
		return protocolv2.ThreadInjectItemsResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) List(ctx context.Context, params protocolv2.ThreadListParams) (protocolv2.ThreadListResponse, error) {
	var response protocolv2.ThreadListResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadList, params, &response); err != nil {
		return protocolv2.ThreadListResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) LoadedList(ctx context.Context, params protocolv2.ThreadLoadedListParams) (protocolv2.ThreadLoadedListResponse, error) {
	var response protocolv2.ThreadLoadedListResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadLoadedList, params, &response); err != nil {
		return protocolv2.ThreadLoadedListResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) MetadataUpdate(ctx context.Context, params protocolv2.ThreadMetadataUpdateParams) (protocolv2.ThreadMetadataUpdateResponse, error) {
	var response protocolv2.ThreadMetadataUpdateResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadMetadataUpdate, params, &response); err != nil {
		return protocolv2.ThreadMetadataUpdateResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) NameSet(ctx context.Context, params protocolv2.ThreadSetNameParams) (protocolv2.ThreadSetNameResponse, error) {
	var response protocolv2.ThreadSetNameResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadNameSet, params, &response); err != nil {
		return protocolv2.ThreadSetNameResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) Read(ctx context.Context, params protocolv2.ThreadReadParams) (protocolv2.ThreadReadResponse, error) {
	var response protocolv2.ThreadReadResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadRead, params, &response); err != nil {
		return protocolv2.ThreadReadResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) RealtimeAppendAudio(ctx context.Context, params protocolv2.ThreadRealtimeAppendAudioParams) (protocolv2.ThreadRealtimeAppendAudioResponse, error) {
	var response protocolv2.ThreadRealtimeAppendAudioResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadRealtimeAppendAudio, params, &response); err != nil {
		return protocolv2.ThreadRealtimeAppendAudioResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) RealtimeAppendText(ctx context.Context, params protocolv2.ThreadRealtimeAppendTextParams) (protocolv2.ThreadRealtimeAppendTextResponse, error) {
	var response protocolv2.ThreadRealtimeAppendTextResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadRealtimeAppendText, params, &response); err != nil {
		return protocolv2.ThreadRealtimeAppendTextResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) RealtimeListVoices(ctx context.Context, params protocolv2.ThreadRealtimeListVoicesParams) (protocolv2.ThreadRealtimeListVoicesResponse, error) {
	var response protocolv2.ThreadRealtimeListVoicesResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadRealtimeListVoices, params, &response); err != nil {
		return protocolv2.ThreadRealtimeListVoicesResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) RealtimeStart(ctx context.Context, params protocolv2.ThreadRealtimeStartParams) (protocolv2.ThreadRealtimeStartResponse, error) {
	var response protocolv2.ThreadRealtimeStartResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadRealtimeStart, params, &response); err != nil {
		return protocolv2.ThreadRealtimeStartResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) RealtimeStop(ctx context.Context, params protocolv2.ThreadRealtimeStopParams) (protocolv2.ThreadRealtimeStopResponse, error) {
	var response protocolv2.ThreadRealtimeStopResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadRealtimeStop, params, &response); err != nil {
		return protocolv2.ThreadRealtimeStopResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) Fork(ctx context.Context, params protocolv2.ThreadForkParams) (protocolv2.ThreadForkResponse, error) {
	var response protocolv2.ThreadForkResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadFork, params, &response); err != nil {
		return protocolv2.ThreadForkResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) GoalClear(ctx context.Context, params protocolv2.ThreadGoalClearParams) (protocolv2.ThreadGoalClearResponse, error) {
	var response protocolv2.ThreadGoalClearResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadGoalClear, params, &response); err != nil {
		return protocolv2.ThreadGoalClearResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) GoalGet(ctx context.Context, params protocolv2.ThreadGoalGetParams) (protocolv2.ThreadGoalGetResponse, error) {
	var response protocolv2.ThreadGoalGetResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadGoalGet, params, &response); err != nil {
		return protocolv2.ThreadGoalGetResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) GoalSet(ctx context.Context, params protocolv2.ThreadGoalSetParams) (protocolv2.ThreadGoalSetResponse, error) {
	var response protocolv2.ThreadGoalSetResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadGoalSet, params, &response); err != nil {
		return protocolv2.ThreadGoalSetResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) IncrementElicitation(ctx context.Context, params protocolv2.ThreadIncrementElicitationParams) (protocolv2.ThreadIncrementElicitationResponse, error) {
	var response protocolv2.ThreadIncrementElicitationResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadIncrementElicitation, params, &response); err != nil {
		return protocolv2.ThreadIncrementElicitationResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) MemoryModeSet(ctx context.Context, params protocolv2.ThreadMemoryModeSetParams) (protocolv2.ThreadMemoryModeSetResponse, error) {
	var response protocolv2.ThreadMemoryModeSetResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadMemoryModeSet, params, &response); err != nil {
		return protocolv2.ThreadMemoryModeSetResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) Rollback(ctx context.Context, params protocolv2.ThreadRollbackParams) (protocolv2.ThreadRollbackResponse, error) {
	var response protocolv2.ThreadRollbackResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadRollback, params, &response); err != nil {
		return protocolv2.ThreadRollbackResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) ShellCommand(ctx context.Context, params protocolv2.ThreadShellCommandParams) (protocolv2.ThreadShellCommandResponse, error) {
	var response protocolv2.ThreadShellCommandResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadShellCommand, params, &response); err != nil {
		return protocolv2.ThreadShellCommandResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) TurnsItemsList(ctx context.Context, params protocolv2.ThreadTurnsItemsListParams) (protocolv2.ThreadTurnsItemsListResponse, error) {
	var response protocolv2.ThreadTurnsItemsListResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadTurnsItemsList, params, &response); err != nil {
		return protocolv2.ThreadTurnsItemsListResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) TurnsList(ctx context.Context, params protocolv2.ThreadTurnsListParams) (protocolv2.ThreadTurnsListResponse, error) {
	var response protocolv2.ThreadTurnsListResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadTurnsList, params, &response); err != nil {
		return protocolv2.ThreadTurnsListResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) Unarchive(ctx context.Context, params protocolv2.ThreadUnarchiveParams) (protocolv2.ThreadUnarchiveResponse, error) {
	var response protocolv2.ThreadUnarchiveResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadUnarchive, params, &response); err != nil {
		return protocolv2.ThreadUnarchiveResponse{}, err
	}
	return response, nil
}

func (f threadsFacade) Unsubscribe(ctx context.Context, params protocolv2.ThreadUnsubscribeParams) (protocolv2.ThreadUnsubscribeResponse, error) {
	var response protocolv2.ThreadUnsubscribeResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodThreadUnsubscribe, params, &response); err != nil {
		return protocolv2.ThreadUnsubscribeResponse{}, err
	}
	return response, nil
}

func (f turnsFacade) Interrupt(ctx context.Context, params protocolv2.TurnInterruptParams) (protocolv2.TurnInterruptResponse, error) {
	var response protocolv2.TurnInterruptResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodTurnInterrupt, params, &response); err != nil {
		return protocolv2.TurnInterruptResponse{}, err
	}
	return response, nil
}

func (f turnsFacade) Start(ctx context.Context, params protocolv2.TurnStartParams) (protocolv2.TurnStartResponse, error) {
	var response protocolv2.TurnStartResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodTurnStart, params, &response); err != nil {
		return protocolv2.TurnStartResponse{}, err
	}
	return response, nil
}

func (f turnsFacade) Steer(ctx context.Context, params protocolv2.TurnSteerParams) (protocolv2.TurnSteerResponse, error) {
	var response protocolv2.TurnSteerResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodTurnSteer, params, &response); err != nil {
		return protocolv2.TurnSteerResponse{}, err
	}
	return response, nil
}

func (f windowsSandboxFacade) Readiness(ctx context.Context) (protocolv2.WindowsSandboxReadinessResponse, error) {
	var response protocolv2.WindowsSandboxReadinessResponse
	if err := f.client.callProtocolNoParams(ctx, protocolv2.MethodWindowsSandboxReadiness, &response); err != nil {
		return protocolv2.WindowsSandboxReadinessResponse{}, err
	}
	return response, nil
}

func (f windowsSandboxFacade) SetupStart(ctx context.Context, params protocolv2.WindowsSandboxSetupStartParams) (protocolv2.WindowsSandboxSetupStartResponse, error) {
	var response protocolv2.WindowsSandboxSetupStartResponse
	if err := f.client.callProtocol(ctx, protocolv2.MethodWindowsSandboxSetupStart, params, &response); err != nil {
		return protocolv2.WindowsSandboxSetupStartResponse{}, err
	}
	return response, nil
}

func (c *client) callProtocol(ctx context.Context, method string, params any, response any) error {
	if c == nil {
		return ErrClientClosed
	}
	if err := c.checkOpen(); err != nil {
		return err
	}
	if err := c.checkProtocolMethodAllowed(method); err != nil {
		return err
	}
	if err := c.checkProtocolParamsAllowed(method, params); err != nil {
		return err
	}
	paramsMap, err := encodeProtocolParams(method, params)
	if err != nil {
		return err
	}
	if _, err := c.callValidated(ctx, method, paramsMap, func(result map[string]any) error {
		return decodeProtocolResponse(method, result, response)
	}); err != nil {
		return err
	}
	return nil
}

func (c *client) callProtocolNoParams(ctx context.Context, method string, response any) error {
	if c == nil {
		return ErrClientClosed
	}
	if err := c.checkOpen(); err != nil {
		return err
	}
	if err := c.checkProtocolMethodAllowed(method); err != nil {
		return err
	}
	if _, err := c.callValidated(ctx, method, nil, func(result map[string]any) error {
		return decodeProtocolResponse(method, result, response)
	}); err != nil {
		return err
	}
	return nil
}

func (c *client) checkProtocolMethodAllowed(method string) error {
	info, ok := protocolv2.LookupMethod(method)
	if !ok {
		return fmt.Errorf("codexsdk: unknown app-server method %q", method)
	}
	if info.Stability == protocolv2.MethodStabilityExperimental && !c.options.Capabilities.ExperimentalAPI {
		return fmt.Errorf("codexsdk: experimental app-server method %q requires ClientCapabilities.ExperimentalAPI", method)
	}
	return nil
}

func (c *client) checkProtocolParamsAllowed(method string, params any) error {
	if c.options.Capabilities.ExperimentalAPI {
		return nil
	}
	switch typed := params.(type) {
	case protocolv2.CommandExecParams:
		return rejectCommandExecExperimentalFields(method, typed)
	case *protocolv2.CommandExecParams:
		if typed == nil {
			return nil
		}
		return rejectCommandExecExperimentalFields(method, *typed)
	case protocolv2.ThreadForkParams:
		return rejectThreadForkExperimentalFields(method, typed)
	case *protocolv2.ThreadForkParams:
		if typed == nil {
			return nil
		}
		return rejectThreadForkExperimentalFields(method, *typed)
	case protocolv2.ThreadResumeParams:
		return rejectThreadResumeExperimentalFields(method, typed)
	case *protocolv2.ThreadResumeParams:
		if typed == nil {
			return nil
		}
		return rejectThreadResumeExperimentalFields(method, *typed)
	case protocolv2.ThreadStartParams:
		return rejectThreadStartExperimentalFields(method, typed)
	case *protocolv2.ThreadStartParams:
		if typed == nil {
			return nil
		}
		return rejectThreadStartExperimentalFields(method, *typed)
	case protocolv2.TurnStartParams:
		return rejectTurnStartExperimentalFields(method, typed)
	case *protocolv2.TurnStartParams:
		if typed == nil {
			return nil
		}
		return rejectTurnStartExperimentalFields(method, *typed)
	case protocolv2.TurnSteerParams:
		return rejectTurnSteerExperimentalFields(method, typed)
	case *protocolv2.TurnSteerParams:
		if typed == nil {
			return nil
		}
		return rejectTurnSteerExperimentalFields(method, *typed)
	default:
		return nil
	}
}

func experimentalFieldError(method, field string) error {
	return fmt.Errorf("codexsdk: experimental field %s.%s requires ClientCapabilities.ExperimentalAPI", method, field)
}

func rejectCommandExecExperimentalFields(method string, params protocolv2.CommandExecParams) error {
	if params.PermissionProfile != nil {
		return experimentalFieldError(method, "permissionProfile")
	}
	return nil
}

func rejectThreadForkExperimentalFields(method string, params protocolv2.ThreadForkParams) error {
	if params.ExcludeTurns != nil {
		return experimentalFieldError(method, "excludeTurns")
	}
	if params.Path != nil {
		return experimentalFieldError(method, "path")
	}
	if params.Permissions != nil {
		return experimentalFieldError(method, "permissions")
	}
	if params.PersistExtendedHistory != nil {
		return experimentalFieldError(method, "persistExtendedHistory")
	}
	return nil
}

func rejectThreadResumeExperimentalFields(method string, params protocolv2.ThreadResumeParams) error {
	if params.ExcludeTurns != nil {
		return experimentalFieldError(method, "excludeTurns")
	}
	if params.History != nil {
		return experimentalFieldError(method, "history")
	}
	if params.Path != nil {
		return experimentalFieldError(method, "path")
	}
	if params.Permissions != nil {
		return experimentalFieldError(method, "permissions")
	}
	if params.PersistExtendedHistory != nil {
		return experimentalFieldError(method, "persistExtendedHistory")
	}
	return nil
}

func rejectThreadStartExperimentalFields(method string, params protocolv2.ThreadStartParams) error {
	if params.DynamicTools != nil {
		return experimentalFieldError(method, "dynamicTools")
	}
	if params.Environments != nil {
		return experimentalFieldError(method, "environments")
	}
	if params.ExperimentalRawEvents != nil {
		return experimentalFieldError(method, "experimentalRawEvents")
	}
	if params.MockExperimentalField != nil {
		return experimentalFieldError(method, "mockExperimentalField")
	}
	if params.Permissions != nil {
		return experimentalFieldError(method, "permissions")
	}
	if params.PersistExtendedHistory != nil {
		return experimentalFieldError(method, "persistExtendedHistory")
	}
	return nil
}

func rejectTurnStartExperimentalFields(method string, params protocolv2.TurnStartParams) error {
	if params.CollaborationMode != nil {
		return experimentalFieldError(method, "collaborationMode")
	}
	if params.Environments != nil {
		return experimentalFieldError(method, "environments")
	}
	if params.Permissions != nil {
		return experimentalFieldError(method, "permissions")
	}
	if params.ResponsesapiClientMetadata != nil {
		return experimentalFieldError(method, "responsesapiClientMetadata")
	}
	return nil
}

func rejectTurnSteerExperimentalFields(method string, params protocolv2.TurnSteerParams) error {
	if params.ResponsesapiClientMetadata != nil {
		return experimentalFieldError(method, "responsesapiClientMetadata")
	}
	return nil
}

func encodeProtocolParams(method string, params any) (map[string]any, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("codexsdk: encode %s params: %w", method, err)
	}
	var paramsMap map[string]any
	if err := json.Unmarshal(raw, &paramsMap); err != nil {
		return nil, fmt.Errorf("codexsdk: encode %s params object: %w", method, err)
	}
	if paramsMap == nil {
		return nil, fmt.Errorf("codexsdk: encode %s params: protocol params must encode to object", method)
	}
	return paramsMap, nil
}

func decodeProtocolResponse(method string, result map[string]any, response any) error {
	if response == nil {
		return errors.New("codexsdk: protocol response target is nil")
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("codexsdk: decode %s response: %w", method, err)
	}
	if err := json.Unmarshal(raw, response); err != nil {
		return fmt.Errorf("codexsdk: decode %s response: %w", method, err)
	}
	return nil
}
