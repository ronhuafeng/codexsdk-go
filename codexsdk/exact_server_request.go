package codexsdk

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ronhuafeng/codexsdk-go/codexsdk/protocolv2"
)

func (c *client) handleExactServerRequest(message map[string]any) {
	id := message["id"]
	raw, err := json.Marshal(message)
	if err != nil {
		c.writeServerRequestError(id, -32602, err)
		c.failClient(err)
		return
	}
	var request protocolv2.ServerRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		failure := fmt.Errorf("codexsdk: decode generated server request: %w", err)
		c.writeServerRequestError(id, -32602, failure)
		c.failClient(failure)
		return
	}
	c.handlerWG.Add(1)
	go func() {
		defer c.handlerWG.Done()
		c.respondToExactServerRequest(id, request)
	}()
}

func (c *client) respondToExactServerRequest(id any, request protocolv2.ServerRequest) {
	response, err := invokeExactServerRequestHandler(c.ctx, c.options.ServerRequestHandler, request)
	if err != nil {
		if c.closingNormally() {
			return
		}
		c.writeServerRequestError(id, -32000, err)
		c.failClient(err)
		return
	}
	if response.kind != request.Kind() || response.value == nil {
		failure := fmt.Errorf("codexsdk: server request %s received mismatched or empty response %s", request.Kind(), response.kind)
		c.writeServerRequestError(id, -32602, failure)
		c.failClient(failure)
		return
	}
	raw, err := json.Marshal(response.value)
	if err != nil {
		failure := fmt.Errorf("codexsdk: encode %s response: %w", request.Kind(), err)
		c.writeServerRequestError(id, -32602, failure)
		c.failClient(failure)
		return
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		failure := fmt.Errorf("codexsdk: decode %s response object: %w", request.Kind(), err)
		c.writeServerRequestError(id, -32602, failure)
		c.failClient(failure)
		return
	}
	if err := c.write(map[string]any{"id": id, "result": result}); err != nil {
		c.failClient(err)
	}
}

func (c *client) closingNormally() bool {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	return c.normalClosing && c.failure == nil
}

func invokeExactServerRequestHandler(ctx context.Context, handler ServerRequestHandler, request protocolv2.ServerRequest) (response ServerRequestResponse, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%w: server request handler panic: %v", ErrHandlerFailed, recovered)
		}
	}()
	response, err = handler(ctx, request)
	if err != nil {
		return ServerRequestResponse{}, fmt.Errorf("%w: %w", ErrHandlerFailed, err)
	}
	return response, nil
}
