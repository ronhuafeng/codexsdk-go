package codexsdk

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ronhuafeng/codexsdk-go/codexsdk/protocolv2"
)

func (c *client) enqueueNotification(notification protocolv2.ServerNotification) error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	if c.closed {
		return nil
	}
	hasHandler := c.options.ServerNotificationHandler != nil
	if hasHandler {
		c.handlerWG.Add(1)
	}
	select {
	case c.notifications <- notification:
		return nil
	default:
		if hasHandler {
			c.handlerWG.Done()
		}
		return ErrNotificationBackpressure
	}
}

func (c *client) notificationDispatcher() {
	defer close(c.dispatcherDone)
	handler := c.options.ServerNotificationHandler
	for {
		select {
		case notification := <-c.notifications:
			if c.ctx.Err() != nil {
				c.endNotificationHandler()
				c.discardAcceptedNotifications()
				return
			}
			if handler != nil && !c.dispatchAcceptedNotification(handler, notification) {
				return
			}
		case <-c.dispatchStop:
			for {
				select {
				case notification := <-c.notifications:
					if handler != nil && !c.dispatchAcceptedNotification(handler, notification) {
						return
					}
				default:
					return
				}
			}
		case <-c.ctx.Done():
			c.discardAcceptedNotifications()
			return
		}
	}
}

func (c *client) dispatchAcceptedNotification(handler ServerNotificationHandler, notification protocolv2.ServerNotification) bool {
	err := invokeNotificationHandler(c.ctx, handler, notification)
	c.endNotificationHandler()
	if err == nil {
		return true
	}
	c.discardAcceptedNotifications()
	c.failClient(err)
	return false
}

func (c *client) beginHandler() (context.Context, bool) {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	if c.closed {
		return nil, false
	}
	c.handlerWG.Add(1)
	return c.callbackContext(), true
}

// callbackContext returns the immutable client-scoped callback context.
// Callers that admit new work must still hold closeMu while adding its WG count.
func (c *client) callbackContext() context.Context {
	if c.handlerCtx != nil {
		return c.handlerCtx
	}
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}

func (c *client) endHandler() {
	c.handlerWG.Done()
}

func (c *client) endNotificationHandler() {
	if c.options.ServerNotificationHandler != nil {
		c.handlerWG.Done()
	}
}

func (c *client) discardAcceptedNotifications() {
	if c.options.ServerNotificationHandler == nil {
		return
	}
	for {
		select {
		case <-c.notifications:
			c.handlerWG.Done()
		default:
			return
		}
	}
}

func invokeNotificationHandler(ctx context.Context, handler ServerNotificationHandler, notification protocolv2.ServerNotification) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%w: notification handler panic: %v", ErrHandlerFailed, recovered)
		}
	}()
	if err := handler(ctx, notification); err != nil {
		return fmt.Errorf("%w: %w", ErrHandlerFailed, err)
	}
	return nil
}

func (c *client) failClient(err error) {
	if err == nil {
		return
	}
	c.closeMu.Lock()
	if c.failure != nil {
		c.closeMu.Unlock()
		return
	}
	c.failure = err
	c.closed = true
	cancel := c.cancel
	c.closeMu.Unlock()
	if cancel != nil {
		cancel()
	}
	c.failAll(err)
	go func() { _ = c.Close() }()
}

func (c *client) shutdown() {
	c.closeMu.Lock()
	failure := c.failure
	c.closed = true
	c.normalClosing = failure == nil
	c.closeMu.Unlock()

	if failure == nil {
		c.failAll(ErrClientClosed)
		if c.dispatchStop != nil {
			close(c.dispatchStop)
		}
	} else if c.cancel != nil {
		c.cancel()
	}
	if c.handlerCancel != nil {
		c.handlerCancel()
	}
	if c.dispatcherDone != nil {
		<-c.dispatcherDone
	}
	if c.cancel != nil {
		c.cancel()
	}
	c.handlerWG.Wait()
	c.writeMu.Lock()
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	c.writeMu.Unlock()
	if c.stdout != nil {
		_ = c.stdout.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Signal(os.Interrupt)
		done := make(chan error, 1)
		go func() { done <- c.cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_ = c.cmd.Process.Kill()
			<-done
		}
	}
}
