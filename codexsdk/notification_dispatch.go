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
	select {
	case c.notifications <- notification:
		return nil
	default:
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
				return
			}
			if handler != nil {
				if err := invokeNotificationHandler(c.ctx, handler, notification); err != nil {
					c.failClient(err)
					return
				}
			}
		case <-c.dispatchStop:
			for {
				select {
				case notification := <-c.notifications:
					if handler != nil {
						if err := invokeNotificationHandler(c.ctx, handler, notification); err != nil {
							c.failClient(err)
							return
						}
					}
				default:
					return
				}
			}
		case <-c.ctx.Done():
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
	if c.dispatcherDone != nil {
		<-c.dispatcherDone
	}
	if c.cancel != nil {
		c.cancel()
	}
	c.handlerWG.Wait()
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
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
