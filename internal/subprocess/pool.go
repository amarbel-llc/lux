package subprocess

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/friedenberg/lux/internal/jsonrpc"
	"github.com/friedenberg/lux/internal/lsp"
)

type LSPState int

const (
	LSPStateIdle LSPState = iota
	LSPStateStarting
	LSPStateRunning
	LSPStateStopping
	LSPStateStopped
	LSPStateFailed
)

func (s LSPState) String() string {
	switch s {
	case LSPStateIdle:
		return "idle"
	case LSPStateStarting:
		return "starting"
	case LSPStateRunning:
		return "running"
	case LSPStateStopping:
		return "stopping"
	case LSPStateStopped:
		return "stopped"
	case LSPStateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

type LSPInstance struct {
	Name         string
	Flake        string
	Args         []string
	State        LSPState
	Process      *Process
	Conn         *jsonrpc.Conn
	Capabilities *lsp.ServerCapabilities
	StartedAt    time.Time
	Error        error

	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

type Pool struct {
	executor  Executor
	instances map[string]*LSPInstance
	mu        sync.RWMutex
	handler   jsonrpc.Handler
}

func NewPool(executor Executor, handler jsonrpc.Handler) *Pool {
	return &Pool{
		executor:  executor,
		instances: make(map[string]*LSPInstance),
		handler:   handler,
	}
}

func (p *Pool) Register(name, flake string, args []string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.instances[name] = &LSPInstance{
		Name:  name,
		Flake: flake,
		Args:  args,
		State: LSPStateIdle,
	}
}

func (p *Pool) Get(name string) (*LSPInstance, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	inst, ok := p.instances[name]
	return inst, ok
}

func (p *Pool) GetOrStart(ctx context.Context, name string, initParams *lsp.InitializeParams) (*LSPInstance, error) {
	p.mu.RLock()
	inst, ok := p.instances[name]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown LSP: %s", name)
	}

	inst.mu.Lock()
	defer inst.mu.Unlock()

	if inst.State == LSPStateRunning {
		return inst, nil
	}

	if inst.State == LSPStateStarting {
		inst.mu.Unlock()
		for {
			time.Sleep(50 * time.Millisecond)
			inst.mu.Lock()
			if inst.State == LSPStateRunning {
				return inst, nil
			}
			if inst.State == LSPStateFailed {
				err := inst.Error
				inst.mu.Unlock()
				return nil, err
			}
			inst.mu.Unlock()
		}
	}

	inst.State = LSPStateStarting
	inst.ctx, inst.cancel = context.WithCancel(ctx)

	binPath, err := p.executor.Build(inst.ctx, inst.Flake)
	if err != nil {
		inst.State = LSPStateFailed
		inst.Error = err
		return nil, fmt.Errorf("building %s: %w", name, err)
	}

	proc, err := p.executor.Execute(inst.ctx, binPath, inst.Args)
	if err != nil {
		inst.State = LSPStateFailed
		inst.Error = err
		return nil, fmt.Errorf("executing %s: %w", name, err)
	}

	inst.Process = proc
	inst.Conn = jsonrpc.NewConn(proc.Stdout, proc.Stdin, p.handler)

	go func() {
		if err := inst.Conn.Run(inst.ctx); err != nil {
			inst.mu.Lock()
			inst.State = LSPStateFailed
			inst.Error = err
			inst.mu.Unlock()
		}
	}()

	if initParams != nil {
		result, err := inst.Conn.Call(inst.ctx, lsp.MethodInitialize, initParams)
		if err != nil {
			inst.State = LSPStateFailed
			inst.Error = err
			proc.Kill()
			return nil, fmt.Errorf("initializing %s: %w", name, err)
		}

		var initResult lsp.InitializeResult
		if err := json.Unmarshal(result, &initResult); err != nil {
			inst.State = LSPStateFailed
			inst.Error = err
			proc.Kill()
			return nil, fmt.Errorf("parsing init result from %s: %w", name, err)
		}

		inst.Capabilities = &initResult.Capabilities

		if err := inst.Conn.Notify(lsp.MethodInitialized, struct{}{}); err != nil {
			inst.State = LSPStateFailed
			inst.Error = err
			proc.Kill()
			return nil, fmt.Errorf("sending initialized to %s: %w", name, err)
		}
	}

	inst.State = LSPStateRunning
	inst.StartedAt = time.Now()
	inst.Error = nil

	return inst, nil
}

func (p *Pool) Stop(name string) error {
	p.mu.RLock()
	inst, ok := p.instances[name]
	p.mu.RUnlock()

	if !ok {
		return fmt.Errorf("unknown LSP: %s", name)
	}

	inst.mu.Lock()
	defer inst.mu.Unlock()

	if inst.State != LSPStateRunning {
		return nil
	}

	inst.State = LSPStateStopping

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if inst.Conn != nil {
		inst.Conn.Call(ctx, lsp.MethodShutdown, nil)
		inst.Conn.Notify(lsp.MethodExit, nil)
		inst.Conn.Close()
	}

	if inst.cancel != nil {
		inst.cancel()
	}

	if inst.Process != nil {
		done := make(chan struct{})
		go func() {
			inst.Process.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			inst.Process.Kill()
		}
	}

	inst.State = LSPStateStopped
	inst.Process = nil
	inst.Conn = nil
	inst.Capabilities = nil

	return nil
}

func (p *Pool) StopAll() {
	p.mu.RLock()
	names := make([]string, 0, len(p.instances))
	for name := range p.instances {
		names = append(names, name)
	}
	p.mu.RUnlock()

	for _, name := range names {
		p.Stop(name)
	}
}

func (p *Pool) Status() []LSPStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var statuses []LSPStatus
	for name, inst := range p.instances {
		inst.mu.RLock()
		status := LSPStatus{
			Name:      name,
			Flake:     inst.Flake,
			State:     inst.State.String(),
			StartedAt: inst.StartedAt,
		}
		if inst.Error != nil {
			status.Error = inst.Error.Error()
		}
		inst.mu.RUnlock()
		statuses = append(statuses, status)
	}

	return statuses
}

type LSPStatus struct {
	Name      string    `json:"name"`
	Flake     string    `json:"flake"`
	State     string    `json:"state"`
	StartedAt time.Time `json:"started_at,omitempty"`
	Error     string    `json:"error,omitempty"`
}

func (inst *LSPInstance) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	inst.mu.RLock()
	defer inst.mu.RUnlock()

	if inst.State != LSPStateRunning {
		return nil, fmt.Errorf("LSP %s is not running", inst.Name)
	}

	return inst.Conn.Call(ctx, method, params)
}

func (inst *LSPInstance) Notify(method string, params any) error {
	inst.mu.RLock()
	defer inst.mu.RUnlock()

	if inst.State != LSPStateRunning {
		return fmt.Errorf("LSP %s is not running", inst.Name)
	}

	return inst.Conn.Notify(method, params)
}
