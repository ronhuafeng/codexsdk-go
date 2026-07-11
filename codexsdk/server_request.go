package codexsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ronhuafeng/codexsdk-go/codexsdk/protocolv2"
)

func (c *client) handleServerRequest(message map[string]any) {
	if c.options.ServerRequestHandler != nil {
		c.handleExactServerRequest(message)
		return
	}
	id := message["id"]
	method, _ := message["method"].(string)
	params, _ := message["params"].(map[string]any)
	if err := validateProtocolServerRequest(message); err != nil {
		req := serverRequestIdentity(method, params)
		err = serverRequestValidationError(id, req, err)
		c.writeServerRequestError(id, -32602, err)
		c.routeServerRequestError(req, err)
		return
	}
	req, err := serverRequestFromMethod(method, params)
	if err != nil {
		c.writeServerRequestError(id, -32602, err)
		c.routeServerRequestError(req, err)
		return
	}
	if isSupportedServerRequest(req) && req.TurnID != "" {
		streamCtx, buffered := c.streamContextOrBufferServerRequest(req.TurnID, rpcServerRequest{id: id, method: method, params: params})
		if buffered {
			return
		}
		go c.respondToServerRequestWithContext(id, method, params, streamCtx)
		return
	}
	go c.respondToServerRequest(id, method, params)
}

func validateProtocolServerRequest(message map[string]any) error {
	raw, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("codexsdk: encode server request for validation: %w", err)
	}
	var typed protocolv2.ServerRequest
	if err := json.Unmarshal(raw, &typed); err != nil {
		return err
	}
	return nil
}

func serverRequestValidationError(id any, req ServerRequest, err error) error {
	if req.Kind == ServerRequestUnsupported {
		unsupported := &unsupportedServerRequestError{
			RequestID: requestIDString(id),
			Method:    req.Method,
			Kind:      req.Kind,
			ThreadID:  req.ThreadID,
			TurnID:    req.TurnID,
			ItemID:    req.ItemID,
		}
		return fmt.Errorf("%s: %w", unsupported.Error(), err)
	}
	parts := []string{"codexsdk: decode ServerRequest"}
	if requestID := requestIDString(id); requestID != "" {
		parts = append(parts, "request_id="+requestID)
	}
	if req.Method != "" {
		parts = append(parts, "method="+req.Method)
	}
	if req.ThreadID != "" {
		parts = append(parts, "thread_id="+req.ThreadID)
	}
	if req.TurnID != "" {
		parts = append(parts, "turn_id="+req.TurnID)
	}
	if req.ItemID != "" {
		parts = append(parts, "item_id="+req.ItemID)
	}
	return fmt.Errorf("%s: %w", strings.Join(parts, " "), err)
}

func requestIDString(id any) string {
	if id == nil {
		return ""
	}
	return fmt.Sprint(id)
}

func (c *client) streamContextOrBufferServerRequest(turnID string, request rpcServerRequest) (context.Context, bool) {
	c.turnMu.Lock()
	defer c.turnMu.Unlock()
	for stream := range c.streams[turnID] {
		if stream.ctx != nil {
			return stream.ctx, false
		}
	}
	if c.pendingServer == nil {
		c.pendingServer = map[string][]rpcServerRequest{}
	}
	c.pendingServer[turnID] = append(c.pendingServer[turnID], request)
	return nil, true
}

func (c *client) respondToServerRequest(id any, method string, params map[string]any) {
	c.respondToServerRequestWithContext(id, method, params, nil)
}

func (c *client) respondToServerRequestWithContext(id any, method string, params map[string]any, parent context.Context) {
	req, err := serverRequestFromMethod(method, params)
	if err != nil {
		c.writeServerRequestError(id, -32602, err)
		c.routeServerRequestError(req, err)
		return
	}
	if !isSupportedServerRequest(req) {
		err := c.unsupportedServerRequest(req)
		c.writeServerRequestError(id, -32601, err)
		c.routeServerRequestError(req, err)
		return
	}

	nilHandler := c.options.LegacyServerRequestHandler == nil
	handlerResponse, handlerErr := c.invokeServerRequestHandler(req, parent)
	if handlerErr != nil {
		c.routeServerRequestError(req, handlerErr)
	}
	result, resultErr := serverRequestResponseResult(req, handlerResponse, handlerErr)
	if resultErr != nil {
		if handlerErr == nil {
			c.routeServerRequestError(req, resultErr)
		}
		c.writeFailClosedServerRequestResultOrError(id, req, resultErr)
		return
	}
	if nilHandler && handlerErr == nil {
		c.routeServerRequestDiagnostic(id, req, "server_request_fail_closed")
	}
	if err := c.writeServerRequestResult(id, req, result); err != nil {
		c.routeServerRequestError(req, err)
		c.writeFailClosedServerRequestResultOrError(id, req, err)
	}
}

func (c *client) respondToServerRequestFailClosed(id any, method string, params map[string]any) {
	req, err := serverRequestFromMethod(method, params)
	if err != nil {
		c.writeServerRequestError(id, -32602, err)
		return
	}
	if !isSupportedServerRequest(req) {
		err := c.unsupportedServerRequest(req)
		c.writeServerRequestError(id, -32601, err)
		return
	}
	c.writeFailClosedServerRequestResultOrError(id, req, fmt.Errorf("codexsdk: %s failed closed", req.Method))
}

func (c *client) writeServerRequestError(id any, code int, err error) {
	_ = c.write(map[string]any{"id": id, "error": map[string]any{"code": code, "message": err.Error()}})
}

func (c *client) writeServerRequestResult(id any, req ServerRequest, result any) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("codexsdk: encode %s response: %w", req.Method, err)
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return fmt.Errorf("codexsdk: encode %s response object: %w", req.Method, err)
	}
	return c.write(map[string]any{"id": id, "result": decoded})
}

func (c *client) writeFailClosedServerRequestResultOrError(id any, req ServerRequest, fallbackErr error) {
	result, ok := failClosedServerRequestResponse(req)
	if !ok {
		c.writeServerRequestError(id, -32000, failClosedServerRequestError(req, fallbackErr))
		return
	}
	if err := c.writeServerRequestResult(id, req, result); err != nil {
		c.writeServerRequestError(id, -32000, err)
	}
}

func failClosedServerRequestError(req ServerRequest, fallbackErr error) error {
	if req.Kind == ServerRequestChatGPTAuthRefresh {
		return fmt.Errorf("codexsdk: %s requires typed ChatGPTAuthTokensRefresh response", req.Method)
	}
	return fallbackErr
}

func (c *client) serverRequestContext(req ServerRequest) context.Context {
	if streamCtx := c.streamContext(req.TurnID); streamCtx != nil {
		return streamCtx
	}
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}

func (c *client) unsupportedServerRequest(req ServerRequest) *unsupportedServerRequestError {
	err := &unsupportedServerRequestError{
		Method:   req.Method,
		Kind:     req.Kind,
		ThreadID: req.ThreadID,
		TurnID:   req.TurnID,
		ItemID:   req.ItemID,
	}
	return err
}

func isSupportedServerRequest(req ServerRequest) bool {
	switch req.Kind {
	case ServerRequestApplyPatchApproval,
		ServerRequestExecCommandApproval,
		ServerRequestCommandApproval,
		ServerRequestFileChangeApproval,
		ServerRequestPermissionsApproval,
		ServerRequestUserInput,
		ServerRequestMCPElicitation,
		ServerRequestToolCall,
		ServerRequestChatGPTAuthRefresh:
		return true
	default:
		return false
	}
}

func (c *client) invokeServerRequestHandler(req ServerRequest, parent context.Context) (response LegacyServerRequestResponse, err error) {
	if c.options.LegacyServerRequestHandler == nil {
		return LegacyServerRequestResponse{}, nil
	}
	if parent == nil {
		parent = c.serverRequestContext(req)
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("codexsdk: server request handler panic: %v", recovered)
		}
	}()
	return c.options.LegacyServerRequestHandler(ctx, req)
}

func (c *client) routeServerRequestError(req ServerRequest, err error) {
	if req.TurnID == "" {
		c.turnMu.Lock()
		c.pendingGlobal = err
		c.turnMu.Unlock()
		c.failAll(err)
		return
	}
	c.turnMu.Lock()
	streams := c.streams[req.TurnID]
	targets := make([]*threadStreamState, 0, len(streams))
	for stream := range streams {
		targets = append(targets, stream)
	}
	if len(targets) == 0 {
		if c.pendingErrors == nil {
			c.pendingErrors = map[string]error{}
		}
		c.pendingErrors[req.TurnID] = err
	}
	c.turnMu.Unlock()
	for _, stream := range targets {
		stream.finishErr(err)
	}
}

func (c *client) routeServerRequestDiagnostic(id any, req ServerRequest, kind string) {
	if req.TurnID == "" {
		return
	}
	ref := DiagnosticRef{
		Kind: kind,
		ID:   requestIDString(id),
	}
	c.turnMu.Lock()
	streams := c.streams[req.TurnID]
	targets := make([]*threadStreamState, 0, len(streams))
	for stream := range streams {
		targets = append(targets, stream)
	}
	c.turnMu.Unlock()
	for _, stream := range targets {
		diagnostic := ref
		stream.send(ThreadEvent{
			Kind:       ThreadEventDiagnostic,
			ThreadID:   req.ThreadID,
			TurnID:     req.TurnID,
			At:         time.Now(),
			Diagnostic: &diagnostic,
		})
	}
}

func serverRequestFromMethod(method string, params map[string]any) (ServerRequest, error) {
	req := serverRequestIdentity(method, params)
	switch method {
	case protocolv2.MethodApplyPatchApproval:
		var typed protocolv2.ApplyPatchApprovalParams
		if err := decodeServerRequestParams(method, params, &typed); err != nil {
			return req, err
		}
		req.ThreadID = typed.ConversationID
		req.ItemID = typed.CallID
		req.ApplyPatchApproval = &typed
		req.Approval = &ApprovalRequest{
			Method:             method,
			ThreadID:           req.ThreadID,
			ItemID:             req.ItemID,
			Reason:             nullableString(typed.Reason),
			AvailableDecisions: []ApprovalDecision{ApprovalAccept, ApprovalAcceptForSession, ApprovalDecline, ApprovalCancel},
		}
	case protocolv2.MethodExecCommandApproval:
		var typed protocolv2.ExecCommandApprovalParams
		if err := decodeServerRequestParams(method, params, &typed); err != nil {
			return req, err
		}
		req.ThreadID = typed.ConversationID
		req.ItemID = typed.CallID
		req.ExecCommandApproval = &typed
		req.Approval = &ApprovalRequest{
			Method:             method,
			ThreadID:           req.ThreadID,
			ItemID:             req.ItemID,
			Reason:             nullableString(typed.Reason),
			CWD:                typed.CWD,
			Command:            append([]string(nil), typed.Command...),
			AvailableDecisions: []ApprovalDecision{ApprovalAccept, ApprovalAcceptForSession, ApprovalDecline, ApprovalCancel},
		}
	case protocolv2.MethodItemCommandExecutionRequestApproval:
		var typed protocolv2.CommandExecutionRequestApprovalParams
		if err := decodeServerRequestParams(method, params, &typed); err != nil {
			return req, err
		}
		req.ThreadID = typed.ThreadID
		req.TurnID = typed.TurnID
		req.ItemID = typed.ItemID
		req.CommandExecutionApproval = &typed
		req.Approval = &ApprovalRequest{
			Method:             method,
			ThreadID:           req.ThreadID,
			TurnID:             req.TurnID,
			ItemID:             req.ItemID,
			Reason:             nullableString(typed.Reason),
			CWD:                nullableString(typed.CWD),
			Command:            singletonCommand(nullableString(typed.Command)),
			AvailableDecisions: commandExecutionApprovalDecisions(typed.AvailableDecisions),
		}
	case protocolv2.MethodItemFileChangeRequestApproval:
		var typed protocolv2.FileChangeRequestApprovalParams
		if err := decodeServerRequestParams(method, params, &typed); err != nil {
			return req, err
		}
		req.ThreadID = typed.ThreadID
		req.TurnID = typed.TurnID
		req.ItemID = typed.ItemID
		req.FileChangeApproval = &typed
		req.Approval = &ApprovalRequest{
			Method:             method,
			ThreadID:           req.ThreadID,
			TurnID:             req.TurnID,
			ItemID:             req.ItemID,
			Reason:             nullableString(typed.Reason),
			AvailableDecisions: []ApprovalDecision{ApprovalAccept, ApprovalAcceptForSession, ApprovalDecline, ApprovalCancel},
		}
	case protocolv2.MethodItemPermissionsRequestApproval:
		var typed protocolv2.PermissionsRequestApprovalParams
		if err := decodeServerRequestParams(method, params, &typed); err != nil {
			return req, err
		}
		req.ThreadID = typed.ThreadID
		req.TurnID = typed.TurnID
		req.ItemID = typed.ItemID
		req.PermissionsApproval = &typed
		req.Approval = &ApprovalRequest{
			Method:             method,
			ThreadID:           req.ThreadID,
			TurnID:             req.TurnID,
			ItemID:             req.ItemID,
			Reason:             nullableString(typed.Reason),
			CWD:                typed.CWD,
			AvailableDecisions: []ApprovalDecision{ApprovalDecline},
		}
	case protocolv2.MethodAccountChatGPTAuthTokensRefresh:
		var typed protocolv2.ChatgptAuthTokensRefreshParams
		if err := decodeServerRequestParams(method, params, &typed); err != nil {
			return req, err
		}
		req.ChatGPTAuthTokensRefresh = &typed
	case protocolv2.MethodItemToolCall:
		var typed protocolv2.DynamicToolCallParams
		if err := decodeServerRequestParams(method, params, &typed); err != nil {
			return req, err
		}
		req.ThreadID = typed.ThreadID
		req.TurnID = typed.TurnID
		req.ItemID = typed.CallID
		req.DynamicToolCall = &typed
	case protocolv2.MethodItemToolRequestUserInput:
		var typed protocolv2.ToolRequestUserInputParams
		if err := decodeServerRequestParams(method, params, &typed); err != nil {
			return req, err
		}
		req.ThreadID = typed.ThreadID
		req.TurnID = typed.TurnID
		req.ItemID = typed.ItemID
		req.ToolRequestUserInput = &typed
	case protocolv2.MethodMCPServerElicitationRequest:
		var typed protocolv2.McpServerElicitationRequestParams
		if err := decodeServerRequestParams(method, params, &typed); err != nil {
			return req, err
		}
		req.ThreadID = typed.ThreadID
		req.TurnID = nullableString(typed.TurnID)
		req.MCPElicitation = &typed
	}
	return req, nil
}

func serverRequestIdentity(method string, params map[string]any) ServerRequest {
	req := ServerRequest{
		Method:   method,
		Kind:     serverRequestKind(method),
		ThreadID: identityString(params, "threadId", "thread_id", "conversationId"),
		TurnID:   identityString(params, "turnId", "turn_id"),
		ItemID:   identityString(params, "itemId", "item_id", "callId"),
	}
	if req.ItemID == "" {
		req.ItemID = stringAt(params, "item", "id")
	}
	return req
}

func serverRequestKind(method string) ServerRequestKind {
	switch method {
	case protocolv2.MethodApplyPatchApproval:
		return ServerRequestApplyPatchApproval
	case protocolv2.MethodExecCommandApproval:
		return ServerRequestExecCommandApproval
	case protocolv2.MethodItemCommandExecutionRequestApproval:
		return ServerRequestCommandApproval
	case protocolv2.MethodItemFileChangeRequestApproval:
		return ServerRequestFileChangeApproval
	case protocolv2.MethodItemPermissionsRequestApproval:
		return ServerRequestPermissionsApproval
	case protocolv2.MethodItemToolRequestUserInput:
		return ServerRequestUserInput
	case protocolv2.MethodMCPServerElicitationRequest:
		return ServerRequestMCPElicitation
	case protocolv2.MethodItemToolCall:
		return ServerRequestToolCall
	case protocolv2.MethodAccountChatGPTAuthTokensRefresh:
		return ServerRequestChatGPTAuthRefresh
	case "attestation/generate":
		return ServerRequestAttestation
	default:
		return ServerRequestUnsupported
	}
}

func decodeServerRequestParams(method string, params map[string]any, target any) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("codexsdk: decode %s params: %w", method, err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("codexsdk: decode %s params: %w", method, err)
	}
	return nil
}

func serverRequestResponseResult(req ServerRequest, response LegacyServerRequestResponse, handlerErr error) (any, error) {
	if handlerErr != nil {
		if result, ok := failClosedServerRequestResponse(req); ok {
			return result, nil
		}
		return nil, handlerErr
	}
	switch req.Kind {
	case ServerRequestApplyPatchApproval:
		if response.ApplyPatchApproval != nil {
			return *response.ApplyPatchApproval, nil
		}
		decision, err := reviewDecisionFromApprovalDecision(response.ApprovalDecision)
		if err != nil {
			return nil, err
		}
		return protocolv2.ApplyPatchApprovalResponse{Decision: decision}, nil
	case ServerRequestExecCommandApproval:
		if response.ExecCommandApproval != nil {
			return *response.ExecCommandApproval, nil
		}
		decision, err := reviewDecisionFromApprovalDecision(response.ApprovalDecision)
		if err != nil {
			return nil, err
		}
		return protocolv2.ExecCommandApprovalResponse{Decision: decision}, nil
	case ServerRequestCommandApproval:
		if response.CommandExecutionApproval != nil {
			return *response.CommandExecutionApproval, nil
		}
		decision, err := commandExecutionDecisionFromApprovalDecision(response.ApprovalDecision)
		if err != nil {
			return nil, err
		}
		return protocolv2.CommandExecutionRequestApprovalResponse{Decision: decision}, nil
	case ServerRequestFileChangeApproval:
		if response.FileChangeApproval != nil {
			return *response.FileChangeApproval, nil
		}
		decision, err := fileChangeDecisionFromApprovalDecision(response.ApprovalDecision)
		if err != nil {
			return nil, err
		}
		return protocolv2.FileChangeRequestApprovalResponse{Decision: decision}, nil
	case ServerRequestPermissionsApproval:
		if response.PermissionsApproval != nil {
			return *response.PermissionsApproval, nil
		}
		switch response.ApprovalDecision {
		case "", ApprovalDecline:
			return protocolv2.PermissionsRequestApprovalResponse{Permissions: protocolv2.GrantedPermissionProfile{}}, nil
		case ApprovalAccept, ApprovalAcceptForSession:
			return nil, fmt.Errorf("codexsdk: %s requires typed PermissionsApproval response to grant permissions", req.Method)
		default:
			return nil, fmt.Errorf("codexsdk: invalid approval decision %q for %s", response.ApprovalDecision, req.Method)
		}
	case ServerRequestChatGPTAuthRefresh:
		if response.ChatGPTAuthTokensRefresh != nil {
			return *response.ChatGPTAuthTokensRefresh, nil
		}
		return nil, fmt.Errorf("codexsdk: %s requires typed ChatGPTAuthTokensRefresh response", req.Method)
	case ServerRequestToolCall:
		if response.DynamicToolCall != nil {
			return *response.DynamicToolCall, nil
		}
		result, _ := failClosedServerRequestResponse(req)
		return result, nil
	case ServerRequestUserInput:
		if response.ToolRequestUserInput != nil {
			return *response.ToolRequestUserInput, nil
		}
		result, _ := failClosedServerRequestResponse(req)
		return result, nil
	case ServerRequestMCPElicitation:
		if response.MCPElicitation != nil {
			return *response.MCPElicitation, nil
		}
		result, _ := failClosedServerRequestResponse(req)
		return result, nil
	default:
		return nil, fmt.Errorf("codexsdk: unsupported server request %s response", req.Method)
	}
}

func failClosedServerRequestResponse(req ServerRequest) (any, bool) {
	switch req.Kind {
	case ServerRequestApplyPatchApproval:
		return protocolv2.ApplyPatchApprovalResponse{Decision: protocolv2.NewReviewDecisionDenied()}, true
	case ServerRequestExecCommandApproval:
		return protocolv2.ExecCommandApprovalResponse{Decision: protocolv2.NewReviewDecisionDenied()}, true
	case ServerRequestCommandApproval:
		return protocolv2.CommandExecutionRequestApprovalResponse{Decision: protocolv2.NewCommandExecutionApprovalDecisionDecline()}, true
	case ServerRequestFileChangeApproval:
		return protocolv2.FileChangeRequestApprovalResponse{Decision: protocolv2.FileChangeApprovalDecisionDecline}, true
	case ServerRequestPermissionsApproval:
		return protocolv2.PermissionsRequestApprovalResponse{Permissions: protocolv2.GrantedPermissionProfile{}}, true
	case ServerRequestToolCall:
		return protocolv2.DynamicToolCallResponse{ContentItems: []protocolv2.DynamicToolCallOutputContentItem{}, Success: false}, true
	case ServerRequestUserInput:
		return protocolv2.ToolRequestUserInputResponse{Answers: map[string]protocolv2.ToolRequestUserInputAnswer{}}, true
	case ServerRequestMCPElicitation:
		return protocolv2.McpServerElicitationRequestResponse{Action: protocolv2.McpServerElicitationActionDecline}, true
	default:
		return nil, false
	}
}

func reviewDecisionFromApprovalDecision(decision ApprovalDecision) (protocolv2.ReviewDecision, error) {
	switch decision {
	case "", ApprovalDecline:
		return protocolv2.NewReviewDecisionDenied(), nil
	case ApprovalAccept:
		return protocolv2.NewReviewDecisionApproved(), nil
	case ApprovalAcceptForSession:
		return protocolv2.NewReviewDecisionApprovedForSession(), nil
	case ApprovalCancel:
		return protocolv2.NewReviewDecisionAbort(), nil
	default:
		return protocolv2.ReviewDecision{}, fmt.Errorf("codexsdk: invalid approval decision %q", decision)
	}
}

func commandExecutionDecisionFromApprovalDecision(decision ApprovalDecision) (protocolv2.CommandExecutionApprovalDecision, error) {
	switch decision {
	case "", ApprovalDecline:
		return protocolv2.NewCommandExecutionApprovalDecisionDecline(), nil
	case ApprovalAccept:
		return protocolv2.NewCommandExecutionApprovalDecisionAccept(), nil
	case ApprovalAcceptForSession:
		return protocolv2.NewCommandExecutionApprovalDecisionAcceptForSession(), nil
	case ApprovalCancel:
		return protocolv2.NewCommandExecutionApprovalDecisionCancel(), nil
	default:
		return protocolv2.CommandExecutionApprovalDecision{}, fmt.Errorf("codexsdk: invalid approval decision %q", decision)
	}
}

func fileChangeDecisionFromApprovalDecision(decision ApprovalDecision) (protocolv2.FileChangeApprovalDecision, error) {
	switch decision {
	case "", ApprovalDecline:
		return protocolv2.FileChangeApprovalDecisionDecline, nil
	case ApprovalAccept:
		return protocolv2.FileChangeApprovalDecisionAccept, nil
	case ApprovalAcceptForSession:
		return protocolv2.FileChangeApprovalDecisionAcceptForSession, nil
	case ApprovalCancel:
		return protocolv2.FileChangeApprovalDecisionCancel, nil
	default:
		return "", fmt.Errorf("codexsdk: invalid approval decision %q", decision)
	}
}

func nullableString[T ~string](value *protocolv2.Nullable[T]) string {
	if value == nil || value.Value == nil {
		return ""
	}
	return string(*value.Value)
}

func singletonCommand(command string) []string {
	if command == "" {
		return nil
	}
	return []string{command}
}

func commandExecutionApprovalDecisions(value *protocolv2.Nullable[[]protocolv2.CommandExecutionApprovalDecision]) []ApprovalDecision {
	if value == nil || value.Value == nil {
		return []ApprovalDecision{ApprovalAccept, ApprovalAcceptForSession, ApprovalDecline, ApprovalCancel}
	}
	decisions := make([]ApprovalDecision, 0, len(*value.Value))
	for _, decision := range *value.Value {
		switch decision.Kind() {
		case protocolv2.CommandExecutionApprovalDecisionKindAccept:
			decisions = append(decisions, ApprovalAccept)
		case protocolv2.CommandExecutionApprovalDecisionKindAcceptForSession:
			decisions = append(decisions, ApprovalAcceptForSession)
		case protocolv2.CommandExecutionApprovalDecisionKindDecline:
			decisions = append(decisions, ApprovalDecline)
		case protocolv2.CommandExecutionApprovalDecisionKindCancel:
			decisions = append(decisions, ApprovalCancel)
		}
	}
	if len(decisions) == 0 {
		return []ApprovalDecision{ApprovalDecline, ApprovalCancel}
	}
	return decisions
}
