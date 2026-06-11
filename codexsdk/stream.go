package codexsdk

import (
	"context"
	"sync"
)

type ThreadStream struct {
	mu      sync.Mutex
	state   *threadStreamState
	current ThreadEvent
}

type threadStreamState struct {
	client   *client
	threadID string
	turnID   string
	events   chan ThreadEvent
	done     chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc

	// notificationOrderMu preserves pending-before-live notification delivery
	// around stream attach. It is not a scheduler or event store.
	notificationOrderMu sync.Mutex
	mu                  sync.Mutex
	result              ThreadRunResult
	hasRes              bool
	err                 error
	closed              bool
	terminal            bool

	inputStats InputStats
	items      []ThreadItem
	usage      Usage
}

func newThreadStream(c *client, threadID string) *ThreadStream {
	parent := context.Background()
	if c != nil && c.ctx != nil {
		parent = c.ctx
	}
	ctx, cancel := context.WithCancel(parent)
	state := &threadStreamState{
		client:   c,
		threadID: threadID,
		events:   make(chan ThreadEvent, 32),
		done:     make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}
	return &ThreadStream{state: state}
}

func (s *ThreadStream) Next(ctx context.Context) bool {
	if s == nil || s.state == nil {
		return false
	}
	select {
	case event := <-s.state.events:
		s.mu.Lock()
		s.current = event
		s.mu.Unlock()
		return true
	default:
	}
	s.state.mu.Lock()
	terminal := s.state.terminal
	s.state.mu.Unlock()
	if terminal {
		return false
	}
	select {
	case event := <-s.state.events:
		s.mu.Lock()
		s.current = event
		s.mu.Unlock()
		return true
	case <-ctx.Done():
		s.state.cancelWait(ctx.Err())
		return false
	case <-s.state.done:
		return false
	}
}

func (s *ThreadStream) Event() ThreadEvent {
	if s == nil {
		return ThreadEvent{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}

func (s *ThreadStream) Result() (ThreadRunResult, bool) {
	if s == nil || s.state == nil {
		return ThreadRunResult{}, false
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	return s.state.result, s.state.hasRes
}

func (s *ThreadStream) Err() error {
	if s == nil || s.state == nil {
		return ErrStreamClosed
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	return s.state.err
}

func (s *ThreadStream) Close() error {
	if s == nil || s.state == nil {
		return nil
	}
	s.state.closeEarly()
	return nil
}

func (s *threadStreamState) setTurnID(turnID string) {
	s.mu.Lock()
	s.turnID = turnID
	s.mu.Unlock()
}

func (s *threadStreamState) send(event ThreadEvent) bool {
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return false
	}
	select {
	case s.events <- event:
		return true
	case <-s.done:
		return false
	}
}

func (s *threadStreamState) isTerminal() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.terminal
}

func (s *threadStreamState) cancelContext() {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *threadStreamState) finishResult(result ThreadRunResult) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.result = result
	s.hasRes = true
	s.closed = true
	s.terminal = true
	cancel := s.cancel
	close(s.done)
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *threadStreamState) finishErr(err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.err = err
	s.closed = true
	s.terminal = true
	client := s.client
	turnID := s.turnID
	cancel := s.cancel
	close(s.done)
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if client != nil && turnID != "" {
		client.unregisterStream(turnID, s)
	}
}

func (s *threadStreamState) closeEarly() {
	s.cancelOwnedTurn(ErrStreamClosed)
}

func (s *threadStreamState) cancelWait(err error) {
	s.cancelOwnedTurn(err)
}

func (s *threadStreamState) cancelOwnedTurn(err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.err = err
	s.closed = true
	s.terminal = true
	threadID := s.threadID
	turnID := s.turnID
	client := s.client
	cancel := s.cancel
	close(s.done)
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if client != nil && turnID != "" {
		client.unregisterStream(turnID, s)
		client.bestEffortInterrupt(threadID, turnID)
	}
}
