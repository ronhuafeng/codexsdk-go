package codexsdk

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ronhuafeng/codexsdk-go/codexsdk/protocolv2"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultClientName  = "codex-go-sdk"
	defaultClientTitle = "Codex Go SDK"
)

type client struct {
	options ClientOptions
	ctx     context.Context
	cancel  context.CancelFunc

	closeMu sync.Mutex
	closed  bool

	writeMu sync.Mutex
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	cmd     *exec.Cmd

	nextID  atomic.Uint64
	pending sync.Map

	turnMu              sync.Mutex
	streams             map[string]map[*threadStreamState]struct{}
	pendingEvents       map[string][]rpcNotification
	pendingErrors       map[string]error
	pendingServer       map[string][]rpcServerRequest
	pendingGlobal       error
	pendingThreadEvents map[string][]rpcNotification
	pendingGlobalEvents []rpcNotification

	readerDone chan struct{}
}

type threadClient struct {
	client  *client
	options ThreadClientOptions
}

type rpcResponse struct {
	result map[string]any
	err    error
}

type pendingCall struct {
	method   string
	response chan rpcResponse
	validate func(map[string]any) error
}

type rpcNotification struct {
	method string
	params map[string]any
}

type rpcServerRequest struct {
	id     any
	method string
	params map[string]any
}

func New(options ClientOptions) (Client, error) {
	normalized, err := validateOptions(options)
	if err != nil {
		return nil, err
	}
	clientCtx, cancel := context.WithCancel(context.Background())
	c := &client{
		options:             normalized,
		ctx:                 clientCtx,
		cancel:              cancel,
		streams:             map[string]map[*threadStreamState]struct{}{},
		pendingEvents:       map[string][]rpcNotification{},
		pendingErrors:       map[string]error{},
		pendingServer:       map[string][]rpcServerRequest{},
		pendingThreadEvents: map[string][]rpcNotification{},
		readerDone:          make(chan struct{}),
	}
	if err := c.start(); err != nil {
		cancel()
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	initializeParamsMap, err := encodeProtocolParams(protocolv2.MethodInitialize, initializeParams(normalized))
	if err != nil {
		_ = c.Close()
		return nil, err
	}
	initializeResult, err := c.call(ctx, protocolv2.MethodInitialize, initializeParamsMap)
	if err != nil {
		_ = c.Close()
		return nil, err
	}
	if err := validateInitializeResponse(initializeResult); err != nil {
		_ = c.Close()
		return nil, err
	}
	if err := c.notify(protocolv2.NewClientNotificationInitialized()); err != nil {
		_ = c.Close()
		return nil, err
	}
	return c, nil
}

func initializeParams(options ClientOptions) protocolv2.InitializeParams {
	capabilities := protocolv2.InitializeCapabilities{}
	if options.Capabilities.ExperimentalAPI {
		capabilities.ExperimentalAPI = Bool(true)
	}
	return protocolv2.InitializeParams{
		Capabilities: protocolv2.Value(capabilities),
		ClientInfo: protocolv2.ClientInfo{
			Name:    defaultString(options.ClientName, defaultClientName),
			Title:   protocolv2.Value(defaultString(options.ClientTitle, defaultClientTitle)),
			Version: "codex-go-sdk-v1",
		},
	}
}

func validateInitializeResponse(result map[string]any) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("codexsdk: initialize response invalid: %w", err)
	}
	var response protocolv2.InitializeResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return fmt.Errorf("codexsdk: initialize response invalid: %w", err)
	}
	return nil
}

func validateOptions(options ClientOptions) (ClientOptions, error) {
	if strings.TrimSpace(options.CWD) == "" {
		return ClientOptions{}, errors.New("codexsdk: ClientOptions.CWD is required")
	}
	info, err := os.Stat(options.CWD)
	if err != nil {
		return ClientOptions{}, fmt.Errorf("codexsdk: invalid ClientOptions.CWD: %w", err)
	}
	if !info.IsDir() {
		return ClientOptions{}, fmt.Errorf("codexsdk: ClientOptions.CWD is not a directory: %s", options.CWD)
	}
	if len(options.Command) == 0 || strings.TrimSpace(options.Command[0]) == "" {
		return ClientOptions{}, errors.New("codexsdk: ClientOptions.Command is required")
	}
	options.Command = append([]string(nil), options.Command...)
	return options, nil
}

func (c *client) ThreadClient(options ThreadClientOptions) ThreadClient {
	return &threadClient{client: c, options: options}
}

func (tc *threadClient) StartThread(ctx context.Context, req StartThreadRequest) (ThreadRunResult, error) {
	stream, err := tc.StartThreadStream(ctx, req)
	if err != nil {
		return ThreadRunResult{}, err
	}
	return drainStream(ctx, stream)
}

func (tc *threadClient) ResumeThread(ctx context.Context, req ResumeThreadRequest) (ThreadRunResult, error) {
	stream, err := tc.ResumeThreadStream(ctx, req)
	if err != nil {
		return ThreadRunResult{}, err
	}
	return drainStream(ctx, stream)
}

func drainStream(ctx context.Context, stream *ThreadStream) (ThreadRunResult, error) {
	defer stream.Close()
	for stream.Next(ctx) {
	}
	if err := stream.Err(); err != nil {
		return ThreadRunResult{}, err
	}
	if result, ok := stream.Result(); ok {
		return result, nil
	}
	return ThreadRunResult{}, errors.New("codexsdk: stream ended without result")
}

func (tc *threadClient) StartThreadStream(ctx context.Context, req StartThreadRequest) (*ThreadStream, error) {
	c := tc.client
	if c == nil {
		return nil, ErrClientClosed
	}
	if err := c.checkOpen(); err != nil {
		return nil, err
	}
	if err := validateOutputSchema(req.OutputSchema); err != nil {
		return nil, err
	}
	model := defaultString(req.Model, tc.options.DefaultModel)
	if strings.TrimSpace(model) == "" {
		return nil, errors.New("codexsdk: StartThreadRequest.Model or ThreadClientOptions.DefaultModel is required")
	}
	if _, err := protocolUserInput(req.Input); err != nil {
		return nil, err
	}
	cwd := defaultString(req.CWD, tc.options.DefaultCWD)
	req.ApprovalPolicy = defaultApprovalPolicy(req.ApprovalPolicy, tc.options.DefaultApprovalPolicy)
	req.ApprovalsReviewer = defaultApprovalsReviewer(req.ApprovalsReviewer, tc.options.DefaultApprovalsReviewer)
	req.Ephemeral = defaultBoolPointer(req.Ephemeral, tc.options.DefaultEphemeral)
	params, err := threadStartProtocolParams(req, model, cwd)
	if err != nil {
		return nil, err
	}
	var started protocolv2.ThreadStartResponse
	if err := c.callProtocol(ctx, protocolv2.MethodThreadStart, params, &started); err != nil {
		return nil, err
	}
	threadID := started.Thread.ID
	if threadID == "" {
		return nil, errors.New("codexsdk: thread/start response missing thread id")
	}
	effort := req.Effort
	if effort == "" {
		effort = tc.options.DefaultEffort
	}
	return c.startTurnStream(ctx, threadID, req.Input, req.OutputSchema, effort, cwd)
}

func (tc *threadClient) ResumeThreadStream(ctx context.Context, req ResumeThreadRequest) (*ThreadStream, error) {
	c := tc.client
	if c == nil {
		return nil, ErrClientClosed
	}
	if err := c.checkOpen(); err != nil {
		return nil, err
	}
	if err := validateOutputSchema(req.OutputSchema); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.ThreadID) == "" {
		return nil, errors.New("codexsdk: ResumeThreadRequest.ThreadID is required")
	}
	if _, err := protocolUserInput(req.Input); err != nil {
		return nil, err
	}
	req.Model = defaultString(req.Model, tc.options.DefaultModel)
	req.CWD = defaultString(req.CWD, tc.options.DefaultCWD)
	req.ApprovalPolicy = defaultApprovalPolicy(req.ApprovalPolicy, tc.options.DefaultApprovalPolicy)
	req.ApprovalsReviewer = defaultApprovalsReviewer(req.ApprovalsReviewer, tc.options.DefaultApprovalsReviewer)
	params, err := threadResumeProtocolParams(req)
	if err != nil {
		return nil, err
	}
	var resumed protocolv2.ThreadResumeResponse
	if err := c.callProtocol(ctx, protocolv2.MethodThreadResume, params, &resumed); err != nil {
		return nil, err
	}
	threadID := resumed.Thread.ID
	if threadID == "" {
		threadID = req.ThreadID
	}
	effort := req.Effort
	if effort == "" {
		effort = tc.options.DefaultEffort
	}
	return c.startTurnStream(ctx, threadID, req.Input, req.OutputSchema, effort, req.CWD)
}

func (tc *threadClient) ForkThread(ctx context.Context, req ForkThreadRequest) (ThreadForkResult, error) {
	c := tc.client
	if c == nil {
		return ThreadForkResult{}, ErrClientClosed
	}
	if err := c.checkOpen(); err != nil {
		return ThreadForkResult{}, err
	}
	if strings.TrimSpace(req.ParentThreadID) == "" {
		return ThreadForkResult{}, errors.New("codexsdk: ForkThreadRequest.ParentThreadID is required")
	}
	req.Model = defaultString(req.Model, tc.options.DefaultModel)
	req.CWD = defaultString(req.CWD, tc.options.DefaultCWD)
	req.ApprovalPolicy = defaultApprovalPolicy(req.ApprovalPolicy, tc.options.DefaultApprovalPolicy)
	req.ApprovalsReviewer = defaultApprovalsReviewer(req.ApprovalsReviewer, tc.options.DefaultApprovalsReviewer)
	req.Ephemeral = defaultBoolPointer(req.Ephemeral, tc.options.DefaultEphemeral)
	params, err := threadForkProtocolParams(req)
	if err != nil {
		return ThreadForkResult{}, err
	}
	var forked protocolv2.ThreadForkResponse
	if err := c.callProtocol(ctx, protocolv2.MethodThreadFork, params, &forked); err != nil {
		return ThreadForkResult{}, err
	}
	threadID := forked.Thread.ID
	if threadID == "" {
		return ThreadForkResult{}, errors.New("codexsdk: thread/fork response missing thread id")
	}
	return ThreadForkResult{
		ThreadID:                 threadID,
		ForkedFromID:             req.ParentThreadID,
		Ephemeral:                req.Ephemeral,
		EffectiveModel:           forked.Model,
		EffectiveModelProvider:   forked.ModelProvider,
		EffectiveReasoningEffort: ReasoningEffort(nullableProtocolReasoningEffort(forked.ReasoningEffort)),
	}, nil
}

func validateOutputSchema(schema protocolv2.OutputSchema) error {
	if !schema.IsValid() {
		return nil
	}
	if _, err := schema.MarshalJSON(); err != nil {
		return fmt.Errorf("codexsdk: invalid OutputSchema: %w", err)
	}
	return nil
}

func defaultApprovalPolicy(request ApprovalPolicy, fallback ApprovalPolicy) ApprovalPolicy {
	if request != "" {
		return request
	}
	return fallback
}

func defaultApprovalsReviewer(request ApprovalsReviewer, fallback ApprovalsReviewer) ApprovalsReviewer {
	if request != "" {
		return request
	}
	return fallback
}

func defaultBoolPointer(request *bool, fallback *bool) *bool {
	if request != nil {
		value := *request
		return &value
	}
	if fallback != nil {
		value := *fallback
		return &value
	}
	return nil
}

func threadStartProtocolParams(req StartThreadRequest, model string, cwd string) (protocolv2.ThreadStartParams, error) {
	approvalPolicy, err := protocolApprovalPolicy(req.ApprovalPolicy)
	if err != nil {
		return protocolv2.ThreadStartParams{}, err
	}
	approvalsReviewer, err := protocolApprovalsReviewer(req.ApprovalsReviewer)
	if err != nil {
		return protocolv2.ThreadStartParams{}, err
	}
	params := protocolv2.ThreadStartParams{}
	if approvalPolicy != nil {
		params.ApprovalPolicy = approvalPolicy
	}
	if approvalsReviewer != nil {
		params.ApprovalsReviewer = approvalsReviewer
	}
	if strings.TrimSpace(cwd) != "" {
		params.CWD = protocolv2.Value(strings.TrimSpace(cwd))
	}
	if req.Ephemeral != nil {
		params.Ephemeral = protocolv2.Value(*req.Ephemeral)
	}
	if strings.TrimSpace(model) != "" {
		params.Model = protocolv2.Value(strings.TrimSpace(model))
	}
	return params, nil
}

func threadResumeProtocolParams(req ResumeThreadRequest) (protocolv2.ThreadResumeParams, error) {
	approvalPolicy, err := protocolApprovalPolicy(req.ApprovalPolicy)
	if err != nil {
		return protocolv2.ThreadResumeParams{}, err
	}
	approvalsReviewer, err := protocolApprovalsReviewer(req.ApprovalsReviewer)
	if err != nil {
		return protocolv2.ThreadResumeParams{}, err
	}
	params := protocolv2.ThreadResumeParams{ThreadID: req.ThreadID}
	if approvalPolicy != nil {
		params.ApprovalPolicy = approvalPolicy
	}
	if approvalsReviewer != nil {
		params.ApprovalsReviewer = approvalsReviewer
	}
	if strings.TrimSpace(req.CWD) != "" {
		params.CWD = protocolv2.Value(strings.TrimSpace(req.CWD))
	}
	if strings.TrimSpace(req.Model) != "" {
		params.Model = protocolv2.Value(strings.TrimSpace(req.Model))
	}
	return params, nil
}

func threadForkProtocolParams(req ForkThreadRequest) (protocolv2.ThreadForkParams, error) {
	approvalPolicy, err := protocolApprovalPolicy(req.ApprovalPolicy)
	if err != nil {
		return protocolv2.ThreadForkParams{}, err
	}
	approvalsReviewer, err := protocolApprovalsReviewer(req.ApprovalsReviewer)
	if err != nil {
		return protocolv2.ThreadForkParams{}, err
	}
	params := protocolv2.ThreadForkParams{
		Ephemeral: req.Ephemeral,
		ThreadID:  req.ParentThreadID,
	}
	if approvalPolicy != nil {
		params.ApprovalPolicy = approvalPolicy
	}
	if approvalsReviewer != nil {
		params.ApprovalsReviewer = approvalsReviewer
	}
	if strings.TrimSpace(req.CWD) != "" {
		params.CWD = protocolv2.Value(strings.TrimSpace(req.CWD))
	}
	if strings.TrimSpace(req.Model) != "" {
		params.Model = protocolv2.Value(strings.TrimSpace(req.Model))
	}
	return params, nil
}

func turnStartProtocolParams(threadID string, input []InputItem, outputSchema protocolv2.OutputSchema, effort ReasoningEffort, cwd string) (protocolv2.TurnStartParams, error) {
	userInput, err := protocolUserInput(input)
	if err != nil {
		return protocolv2.TurnStartParams{}, err
	}
	params := protocolv2.TurnStartParams{
		Input:    userInput,
		ThreadID: threadID,
	}
	if strings.TrimSpace(cwd) != "" {
		params.CWD = protocolv2.Value(strings.TrimSpace(cwd))
	}
	if effort != "" {
		params.Effort = protocolv2.Value(protocolv2.ReasoningEffort(effort))
	}
	if outputSchema.IsValid() {
		params.OutputSchema = &outputSchema
	}
	return params, nil
}

func protocolUserInput(input []InputItem) ([]protocolv2.UserInput, error) {
	out := make([]protocolv2.UserInput, 0, len(input))
	emptyTextElements := []protocolv2.TextElement{}
	for i, item := range input {
		switch item.Type {
		case InputItemText:
			out = append(out, protocolv2.NewUserInputText(protocolv2.UserInputText{
				Text:         item.Text,
				TextElements: &emptyTextElements,
			}))
		case InputItemFile:
			if strings.TrimSpace(item.Path) == "" {
				return nil, fmt.Errorf("codexsdk: InputItem[%d].Path is required for file input", i)
			}
			out = append(out, protocolv2.NewUserInputMention(protocolv2.UserInputMention{
				Name: filepath.Base(item.Path),
				Path: item.Path,
			}))
		default:
			return nil, fmt.Errorf("codexsdk: unsupported InputItem[%d].Type %q", i, item.Type)
		}
	}
	return out, nil
}

func protocolApprovalPolicy(value ApprovalPolicy) (*protocolv2.Nullable[protocolv2.AskForApproval], error) {
	switch value {
	case "":
		return nil, nil
	case ApprovalPolicyUntrusted:
		return protocolv2.Value(protocolv2.NewAskForApprovalUntrusted()), nil
	case ApprovalPolicyOnFailure:
		return nil, fmt.Errorf("codexsdk: unsupported ApprovalPolicy %q", value)
	case ApprovalPolicyOnRequest:
		return protocolv2.Value(protocolv2.NewAskForApprovalOnRequest()), nil
	case ApprovalPolicyNever:
		return protocolv2.Value(protocolv2.NewAskForApprovalNever()), nil
	default:
		return nil, fmt.Errorf("codexsdk: unsupported ApprovalPolicy %q", value)
	}
}

func protocolApprovalsReviewer(value ApprovalsReviewer) (*protocolv2.Nullable[protocolv2.ApprovalsReviewer], error) {
	switch value {
	case "":
		return nil, nil
	case ApprovalsReviewerUser, ApprovalsReviewerAutoReview, ApprovalsReviewerGuardianSubagent:
		return protocolv2.Value(protocolv2.ApprovalsReviewer(value)), nil
	default:
		return nil, fmt.Errorf("codexsdk: unsupported ApprovalsReviewer %q", value)
	}
}

func nullableProtocolReasoningEffort(value *protocolv2.Nullable[protocolv2.ReasoningEffort]) string {
	if value == nil || value.Value == nil {
		return ""
	}
	return string(*value.Value)
}

func (c *client) startTurnStream(ctx context.Context, threadID string, input []InputItem, outputSchema protocolv2.OutputSchema, effort ReasoningEffort, cwd string) (*ThreadStream, error) {
	stream := newThreadStream(c, threadID)
	stream.state.inputStats = inputStats(input)
	params, err := turnStartProtocolParams(threadID, input, outputSchema, effort, cwd)
	if err != nil {
		stream.state.cancelContext()
		return nil, err
	}
	var started protocolv2.TurnStartResponse
	if err := c.callProtocol(ctx, protocolv2.MethodTurnStart, params, &started); err != nil {
		stream.state.cancelContext()
		return nil, err
	}
	turnID := started.Turn.ID
	if turnID == "" {
		stream.state.cancelContext()
		return nil, errors.New("codexsdk: turn/start response missing turn id")
	}
	stream.state.setTurnID(turnID)
	stream.state.send(ThreadEvent{Kind: ThreadEventStarted, ThreadID: threadID, TurnID: turnID, At: time.Now()})
	stream.state.notificationOrderMu.Lock()
	pending, serverRequests, pendingErr := c.attachStreamAndDrainPending(turnID, stream.state)
	for _, notification := range pending {
		stream.state.handleNotificationLocked(notification.method, notification.params)
	}
	if pendingErr != nil {
		stream.state.finishErr(pendingErr)
	}
	terminal := stream.state.isTerminal()
	stream.state.notificationOrderMu.Unlock()
	if terminal {
		for _, request := range serverRequests {
			go c.respondToServerRequestFailClosed(request.id, request.method, request.params)
		}
		return stream, nil
	}
	for _, request := range serverRequests {
		go c.respondToServerRequestWithContext(request.id, request.method, request.params, stream.state.ctx)
	}
	return stream, nil
}

func (c *client) Close() error {
	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return nil
	}
	c.closed = true
	cancel := c.cancel
	stdin := c.stdin
	stdout := c.stdout
	cmd := c.cmd
	c.closeMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if stdin != nil {
		_ = stdin.Close()
	}
	if stdout != nil {
		_ = stdout.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Signal(os.Interrupt)
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	}
	c.failAll(ErrClientClosed)
	return nil
}

func (c *client) checkOpen() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	if c.closed {
		return ErrClientClosed
	}
	return nil
}

func (c *client) start() error {
	command := c.options.Command
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = c.options.CWD
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	c.stdin = stdin
	c.stdout = stdout
	c.cmd = cmd
	go c.drainStderr(stderr)
	go c.readLoop(stdout)
	return nil
}

func (c *client) call(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	return c.callValidated(ctx, method, params, nil)
}

func (c *client) callValidated(ctx context.Context, method string, params map[string]any, validate func(map[string]any) error) (map[string]any, error) {
	if err := c.checkOpen(); err != nil {
		return nil, err
	}
	id := "go-sdk-" + strconv.FormatUint(c.nextID.Add(1), 10)
	ch := make(chan rpcResponse, 1)
	c.pending.Store(id, pendingCall{method: method, response: ch, validate: validate})
	payload := map[string]any{"id": id, "method": method}
	if params != nil {
		payload["params"] = params
	}
	if err := c.write(payload); err != nil {
		c.pending.Delete(id)
		return nil, err
	}
	select {
	case response := <-ch:
		return response.result, response.err
	case <-ctx.Done():
		c.pending.Delete(id)
		return nil, ctx.Err()
	}
}

func (c *client) notify(payload any) error {
	if err := c.checkOpen(); err != nil {
		return err
	}
	return c.write(payload)
}

func (c *client) write(payload any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if c.stdin == nil {
		return errors.New("codexsdk: app-server stdin is not available")
	}
	_, err = c.stdin.Write(append(raw, '\n'))
	return err
}

func (c *client) readLoop(stdout io.Reader) {
	defer close(c.readerDone)
	reader := bufio.NewReader(stdout)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) == 0 && err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			c.failAll(err)
			return
		}
		if err := validateJSONRPCEnvelope(line); err != nil {
			sum := sha256.Sum256(line)
			c.failAll(fmt.Errorf("codexsdk: invalid app-server JSON-RPC line bytes=%d sha256=%s: %w", len(line), hex.EncodeToString(sum[:]), err))
			return
		}
		var message map[string]any
		if err := json.Unmarshal(line, &message); err != nil {
			sum := sha256.Sum256(line)
			c.failAll(fmt.Errorf("codexsdk: invalid app-server JSON-RPC line bytes=%d sha256=%s: %w", len(line), hex.EncodeToString(sum[:]), err))
			return
		}
		c.handleMessage(message)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			c.failAll(err)
			return
		}
	}
	if c.isClosed() {
		c.failAll(ErrClientClosed)
		return
	}
	c.failAll(errors.New("codexsdk: app-server closed stdout"))
}

func (c *client) handleMessage(message map[string]any) {
	_, hasID := message["id"]
	method, hasMethod := message["method"].(string)
	if hasID && hasMethod {
		c.handleServerRequest(message)
		return
	}
	if hasID {
		c.routeResponse(message)
		return
	}
	if hasMethod {
		params, _ := message["params"].(map[string]any)
		c.routeNotification(rpcNotification{method: method, params: params})
	}
}

func (c *client) routeResponse(message map[string]any) {
	id := fmt.Sprint(message["id"])
	raw, ok := c.pending.LoadAndDelete(id)
	if !ok {
		return
	}
	pending := raw.(pendingCall)
	if rawError := message["error"]; rawError != nil {
		pending.response <- rpcResponse{err: fmt.Errorf("codexsdk: app-server error from %s: %s", pending.method, compactJSON(rawError))}
		return
	}
	result, _ := message["result"].(map[string]any)
	if result == nil {
		pending.response <- rpcResponse{err: fmt.Errorf("codexsdk: app-server %s response missing object result", pending.method)}
		return
	}
	if pending.validate != nil {
		if err := pending.validate(result); err != nil {
			pending.response <- rpcResponse{err: err}
			return
		}
	}
	pending.response <- rpcResponse{result: result}
}

func (c *client) routeNotification(notification rpcNotification) {
	if err := validateStreamNotification(notification.method, notification.params); err != nil {
		c.routeNotificationError(notification, err)
		return
	}
	turnID := turnIDFromNotification(notification.method, notification.params)
	if turnID == "" {
		c.routeNoTurnNotification(notification)
		return
	}
	c.turnMu.Lock()
	streams := c.streams[turnID]
	if len(streams) == 0 {
		c.pendingEvents[turnID] = append(c.pendingEvents[turnID], notification)
		c.turnMu.Unlock()
		return
	}
	targets := make([]*threadStreamState, 0, len(streams))
	for stream := range streams {
		targets = append(targets, stream)
	}
	c.turnMu.Unlock()
	for _, stream := range targets {
		stream.handleNotification(notification.method, notification.params)
	}
}

func (c *client) routeNoTurnNotification(notification rpcNotification) {
	if !supportedNoTurnNotification(notification.method) {
		// validateStreamNotification already proved this is a known app-server
		// notification. ThreadClient only projects a narrow no-turn event set;
		// other known background notifications are not stream events.
		return
	}
	threadID := threadIDFromNotification(notification.params)
	c.turnMu.Lock()
	targets := make([]*threadStreamState, 0)
	for _, streams := range c.streams {
		for stream := range streams {
			if threadID == "" || stream.threadID == threadID {
				targets = append(targets, stream)
			}
		}
	}
	if len(targets) == 0 {
		if threadID != "" {
			c.pendingThreadEvents[threadID] = append(c.pendingThreadEvents[threadID], notification)
		} else {
			c.pendingGlobalEvents = append(c.pendingGlobalEvents, notification)
		}
	}
	c.turnMu.Unlock()
	for _, stream := range targets {
		stream.handleNotification(notification.method, notification.params)
	}
}

func (c *client) routeNotificationError(notification rpcNotification, err error) {
	turnID := turnIDFromNotification(notification.method, notification.params)
	if turnID != "" {
		c.turnMu.Lock()
		streams := c.streams[turnID]
		targets := make([]*threadStreamState, 0, len(streams))
		for stream := range streams {
			targets = append(targets, stream)
		}
		if len(targets) == 0 {
			if c.pendingErrors == nil {
				c.pendingErrors = map[string]error{}
			}
			c.pendingErrors[turnID] = err
		}
		c.turnMu.Unlock()
		for _, stream := range targets {
			stream.finishErr(err)
		}
		return
	}

	threadID := threadIDFromNotification(notification.params)
	c.turnMu.Lock()
	targets := make([]*threadStreamState, 0)
	for _, streams := range c.streams {
		for stream := range streams {
			if threadID == "" || stream.threadID == threadID {
				targets = append(targets, stream)
			}
		}
	}
	if len(targets) == 0 {
		c.pendingGlobal = err
	}
	c.turnMu.Unlock()
	if len(targets) == 0 {
		c.failAll(err)
		return
	}
	for _, stream := range targets {
		stream.finishErr(err)
	}
}

func validateStreamNotification(method string, params map[string]any) error {
	raw, err := json.Marshal(map[string]any{
		"method": method,
		"params": params,
	})
	if err != nil {
		return fmt.Errorf("codexsdk: decode %s notification: %w", method, err)
	}
	var typed protocolv2.ServerNotification
	if err := json.Unmarshal(raw, &typed); err != nil {
		return fmt.Errorf("codexsdk: decode %s notification: %w", method, err)
	}
	return nil
}

func decodeNotificationParams(method string, params map[string]any, target any) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("codexsdk: decode %s notification: %w", method, err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("codexsdk: decode %s notification: %w", method, err)
	}
	return nil
}

func (c *client) attachStreamAndDrainPending(turnID string, stream *threadStreamState) ([]rpcNotification, []rpcServerRequest, error) {
	c.turnMu.Lock()
	defer c.turnMu.Unlock()
	if c.streams[turnID] == nil {
		c.streams[turnID] = map[*threadStreamState]struct{}{}
	}
	c.streams[turnID][stream] = struct{}{}
	pending := append([]rpcNotification(nil), c.pendingGlobalEvents...)
	c.pendingGlobalEvents = nil
	pending = append(pending, c.pendingThreadEvents[stream.threadID]...)
	delete(c.pendingThreadEvents, stream.threadID)
	pending = append(pending, c.pendingEvents[turnID]...)
	delete(c.pendingEvents, turnID)
	pendingErr := c.pendingErrors[turnID]
	delete(c.pendingErrors, turnID)
	globalErr := c.pendingGlobal
	c.pendingGlobal = nil
	if pendingErr == nil {
		pendingErr = globalErr
	} else if globalErr != nil {
		pendingErr = errors.Join(pendingErr, globalErr)
	}
	serverRequests := append([]rpcServerRequest(nil), c.pendingServer[turnID]...)
	delete(c.pendingServer, turnID)
	return pending, serverRequests, pendingErr
}

func (c *client) unregisterStream(turnID string, stream *threadStreamState) {
	c.turnMu.Lock()
	defer c.turnMu.Unlock()
	if c.streams[turnID] == nil {
		return
	}
	delete(c.streams[turnID], stream)
	if len(c.streams[turnID]) == 0 {
		delete(c.streams, turnID)
	}
}

func (c *client) streamContext(turnID string) context.Context {
	if turnID == "" {
		return nil
	}
	c.turnMu.Lock()
	defer c.turnMu.Unlock()
	for stream := range c.streams[turnID] {
		if stream.ctx != nil {
			return stream.ctx
		}
	}
	return nil
}

func (c *client) failAll(err error) {
	c.pending.Range(func(key, value any) bool {
		c.pending.Delete(key)
		value.(pendingCall).response <- rpcResponse{err: err}
		return true
	})
	c.turnMu.Lock()
	var streams []*threadStreamState
	for _, byTurn := range c.streams {
		for stream := range byTurn {
			streams = append(streams, stream)
		}
	}
	c.turnMu.Unlock()
	for _, stream := range streams {
		stream.finishErr(err)
	}
}

func (c *client) bestEffortInterrupt(threadID, turnID string) {
	go func() {
		parent := c.ctx
		if parent == nil {
			parent = context.Background()
		}
		ctx, cancel := context.WithTimeout(parent, time.Second)
		defer cancel()
		var response protocolv2.TurnInterruptResponse
		_ = c.callProtocol(ctx, protocolv2.MethodTurnInterrupt, protocolv2.TurnInterruptParams{
			ThreadID: threadID,
			TurnID:   turnID,
		}, &response)
	}()
}

func (c *client) isClosed() bool {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	return c.closed
}

func (c *client) drainStderr(stderr io.Reader) {
	_, _ = io.Copy(io.Discard, stderr)
}

func turnIDFromNotification(method string, params map[string]any) string {
	if params == nil {
		return ""
	}
	if turnID, _ := params["turnId"].(string); turnID != "" {
		return turnID
	}
	if turn, _ := params["turn"].(map[string]any); turn != nil {
		if turnID, _ := turn["id"].(string); turnID != "" {
			return turnID
		}
	}
	if method == "turn/completed" {
		return stringAt(params, "turn", "id")
	}
	if method == "error" {
		return stringAt(params, "error", "turnId")
	}
	return ""
}

func threadIDFromNotification(params map[string]any) string {
	if params == nil {
		return ""
	}
	return identityString(params, "threadId", "thread_id")
}

func supportedNoTurnNotification(method string) bool {
	switch method {
	case "configWarning", "thread/configWarning", "model/rerouted", "thread/modelRerouted", "model/verification", "thread/modelVerification", "error", "turn/error":
		return true
	default:
		return false
	}
}

func (s *threadStreamState) handleNotification(method string, params map[string]any) {
	s.notificationOrderMu.Lock()
	defer s.notificationOrderMu.Unlock()
	s.handleNotificationLocked(method, params)
}

func (s *threadStreamState) handleNotificationLocked(method string, params map[string]any) {
	if params == nil {
		return
	}
	switch method {
	case "item/agentMessage/delta":
		delta := firstString(params, "delta", "text")
		if delta == "" {
			delta = stringAt(params, "message", "delta")
		}
		if delta == "" {
			return
		}
		s.send(ThreadEvent{
			Kind:        ThreadEventOutputDelta,
			ThreadID:    s.threadID,
			TurnID:      s.turnID,
			At:          time.Now(),
			OutputDelta: delta,
		})
	case "item/completed":
		item, _ := params["item"].(map[string]any)
		threadItem := threadItemFromMap(item)
		if threadItem.Type != "" {
			s.mu.Lock()
			s.items = append(s.items, threadItem)
			s.mu.Unlock()
		}
	case "thread/tokenUsage/updated":
		usage, ok := usageFromNotification(params)
		if !ok {
			return
		}
		s.mu.Lock()
		s.usage = usage
		s.mu.Unlock()
		usageCopy := usage
		s.send(ThreadEvent{Kind: ThreadEventUsage, ThreadID: s.threadID, TurnID: s.turnID, At: time.Now(), Usage: &usageCopy})
	case "error", "turn/error":
		warning := warningFromParams(params)
		s.send(ThreadEvent{Kind: ThreadEventTurnWarning, ThreadID: s.threadID, TurnID: s.turnID, At: time.Now(), TurnWarning: &warning})
	case "turn/completed":
		s.handleTurnCompleted(params)
	case "model/rerouted", "thread/modelRerouted":
		model := ModelEvent{
			FromModel: firstString(params, "fromModel"),
			ToModel:   firstString(params, "toModel"),
			Reason:    firstString(params, "reason"),
		}
		s.send(ThreadEvent{Kind: ThreadEventModelRerouted, ThreadID: s.threadID, TurnID: s.turnID, At: time.Now(), Model: &model})
	case "model/verification", "thread/modelVerification":
		model := ModelEvent{Verifications: stringSlice(params["verifications"])}
		s.send(ThreadEvent{Kind: ThreadEventModelVerification, ThreadID: s.threadID, TurnID: s.turnID, At: time.Now(), Model: &model})
	case "configWarning", "thread/configWarning":
		warning := WarningEvent{
			Summary: firstString(params, "summary"),
			Details: firstString(params, "details"),
			Path:    firstString(params, "path"),
		}
		s.send(ThreadEvent{Kind: ThreadEventConfigWarning, ThreadID: s.threadID, At: time.Now(), Warning: &warning})
	}
}

func (s *threadStreamState) handleTurnCompleted(params map[string]any) {
	turn, _ := params["turn"].(map[string]any)
	if turn == nil {
		s.finishErr(errors.New("codexsdk: turn/completed missing turn object"))
		return
	}
	status := firstString(turn, "status")
	turnID := firstString(turn, "id")
	if turnID == "" {
		turnID = s.turnID
	}
	s.mu.Lock()
	for _, item := range itemsFromTurn(turn) {
		threadItem := threadItemFromMap(item)
		if threadItem.Type != "" {
			s.items = append(s.items, threadItem)
		}
	}
	items := append([]ThreadItem(nil), s.items...)
	usage := s.usage
	inputStats := s.inputStats
	s.mu.Unlock()
	switch status {
	case "completed":
	case "failed":
		s.finishErr(&turnError{
			ThreadID: s.threadID,
			TurnID:   turnID,
			Status:   status,
			Code:     turnErrorCode(turn),
			Message:  turnErrorMessage(turn),
		})
		return
	case "interrupted":
		s.finishErr(&turnInterruptedError{ThreadID: s.threadID, TurnID: turnID, Status: status})
		return
	default:
		s.finishErr(fmt.Errorf("codexsdk: turn/completed received non-completed turn status %q", status))
		return
	}
	final, ok := finalResponseFromItems(items)
	if !ok {
		s.finishErr(errors.New("codexsdk: turn completed without final_answer agent message"))
		return
	}
	result := ThreadRunResult{
		ThreadID:                 s.threadID,
		TurnID:                   turnID,
		FinalResponse:            final,
		Items:                    items,
		EffectiveModel:           firstString(turn, "effectiveModel", "model"),
		EffectiveModelProvider:   firstString(turn, "effectiveModelProvider", "modelProvider"),
		EffectiveReasoningEffort: ReasoningEffort(firstString(turn, "effectiveReasoningEffort", "reasoningEffort", "effort")),
		Usage:                    usage,
		InputStats:               inputStats,
	}
	eventResult := result
	s.send(ThreadEvent{Kind: ThreadEventCompleted, ThreadID: s.threadID, TurnID: turnID, At: time.Now(), Result: &eventResult})
	s.finishResult(result)
	if s.client != nil {
		s.client.unregisterStream(turnID, s)
	}
}

func threadItemFromMap(item map[string]any) ThreadItem {
	if item == nil {
		return ThreadItem{}
	}
	return ThreadItem{
		ID:    firstString(item, "id"),
		Type:  firstString(item, "type"),
		Text:  firstString(item, "text"),
		Phase: firstString(item, "phase"),
	}
}

func finalResponseFromItems(items []ThreadItem) (string, bool) {
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		if item.Type == "agentMessage" && item.Phase == "final_answer" && item.Text != "" {
			return item.Text, true
		}
	}
	return "", false
}

func itemsFromTurn(turn map[string]any) []map[string]any {
	rawItems, _ := turn["items"].([]any)
	out := make([]map[string]any, 0, len(rawItems))
	for _, raw := range rawItems {
		item, _ := raw.(map[string]any)
		if item != nil {
			out = append(out, item)
		}
	}
	return out
}

func usageFromNotification(params map[string]any) (Usage, bool) {
	var typed protocolv2.ThreadTokenUsageUpdatedNotification
	if err := decodeNotificationParams(protocolv2.MethodThreadTokenUsageUpdated, params, &typed); err != nil {
		return Usage{}, false
	}
	breakdown := typed.TokenUsage.Last
	return Usage{
		InputTokens:           int(breakdown.InputTokens),
		CachedInputTokens:     int(breakdown.CachedInputTokens),
		OutputTokens:          int(breakdown.OutputTokens),
		ReasoningOutputTokens: int(breakdown.ReasoningOutputTokens),
	}, true
}

func warningFromParams(params map[string]any) TurnWarningEvent {
	errorObj, _ := params["error"].(map[string]any)
	if errorObj == nil {
		errorObj = params
	}
	return TurnWarningEvent{
		Code:      errorCode(errorObj),
		Message:   firstString(errorObj, "message"),
		WillRetry: boolFromAny(firstValue(params, "willRetry", "will_retry")) || boolFromAny(firstValue(errorObj, "willRetry", "will_retry")),
	}
}

func turnErrorMessage(turn map[string]any) string {
	errObj, _ := turn["error"].(map[string]any)
	return firstString(errObj, "message")
}

func turnErrorCode(turn map[string]any) string {
	errObj, _ := turn["error"].(map[string]any)
	return errorCode(errObj)
}

func errorCode(errObj map[string]any) string {
	if code := firstString(errObj, "code"); code != "" {
		return code
	}
	switch info := errObj["codexErrorInfo"].(type) {
	case string:
		return info
	case map[string]any:
		if code := firstString(info, "code", "type"); code != "" {
			return code
		}
		for key := range info {
			return key
		}
	}
	return ""
}

func stringAt(value map[string]any, path ...string) string {
	var current any = value
	for _, key := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = obj[key]
	}
	text, _ := current.(string)
	return strings.TrimSpace(text)
}

func firstString(obj map[string]any, keys ...string) string {
	for _, key := range keys {
		if text, _ := obj[key].(string); strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func firstValue(obj map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := obj[key]; ok {
			return value
		}
	}
	return nil
}

func identityString(obj map[string]any, keys ...string) string {
	for _, key := range keys {
		if text, _ := obj[key].(string); text != "" {
			return text
		}
	}
	return ""
}

func nestedValue(obj map[string]any, path ...string) any {
	var current any = obj
	for _, key := range path {
		typed, _ := current.(map[string]any)
		if typed == nil {
			return nil
		}
		current = typed[key]
	}
	return current
}

func stringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	case string:
		if typed != "" {
			return []string{typed}
		}
	}
	return nil
}

func intFromAny(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil {
			return int(parsed)
		}
	}
	return 0
}

func boolFromAny(value any) bool {
	typed, _ := value.(bool)
	return typed
}

func compactJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(raw)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
