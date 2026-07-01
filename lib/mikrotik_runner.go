package lib

import (
	"context"
	"errors"
	"strings"
	"time"
)

type MikrotikCommandResult struct {
	Items      []map[string]string
	Command    string
	Duration   time.Duration
	Success    bool
	ErrorCode  string
	Err        error
	ExecutedAt time.Time
}

type MikrotikRunner struct {
	client         *MikrotikClient
	commandTimeout time.Duration
}

func NewMikrotikRunner(host string, port int, useTLS bool, connectTimeout time.Duration, commandTimeout time.Duration) (*MikrotikRunner, error) {
	if commandTimeout <= 0 {
		commandTimeout = connectTimeout
	}
	client, err := NewMikrotikClient(host, port, useTLS, connectTimeout)
	if err != nil {
		return nil, err
	}
	return &MikrotikRunner{
		client:         client,
		commandTimeout: commandTimeout,
	}, nil
}

func (r *MikrotikRunner) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

func (r *MikrotikRunner) Login(ctx context.Context, username string, password string) *MikrotikCommandResult {
	if r == nil || r.client == nil {
		return commandFailure("/login", errors.New("mikrotik runner is not connected"), 0)
	}
	if err := ctx.Err(); err != nil {
		return commandFailure("/login", err, 0)
	}

	start := time.Now()
	timeout := r.timeoutFromContext(ctx)
	err := r.client.LoginWithTimeout(username, password, timeout)
	return buildCommandResult("/login", nil, start, err)
}

func (r *MikrotikRunner) Run(ctx context.Context, words ...string) *MikrotikCommandResult {
	command := ""
	if len(words) > 0 {
		command = words[0]
	}
	if r == nil || r.client == nil {
		return commandFailure(command, errors.New("mikrotik runner is not connected"), 0)
	}
	if err := ctx.Err(); err != nil {
		return commandFailure(command, err, 0)
	}

	start := time.Now()
	timeout := r.timeoutFromContext(ctx)
	items, err := r.client.RunWithTimeout(timeout, words...)
	return buildCommandResult(command, items, start, err)
}

func (r *MikrotikRunner) RunReadOnly(ctx context.Context, maxAttempts int, words ...string) *MikrotikCommandResult {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	var last *MikrotikCommandResult
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		last = r.Run(ctx, words...)
		if last.Success || ctx.Err() != nil {
			return last
		}
	}
	return last
}

func (r *MikrotikRunner) timeoutFromContext(ctx context.Context) time.Duration {
	timeout := r.commandTimeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining > 0 && (timeout <= 0 || remaining < timeout) {
			timeout = remaining
		}
	}
	return timeout
}

func buildCommandResult(command string, items []map[string]string, start time.Time, err error) *MikrotikCommandResult {
	result := &MikrotikCommandResult{
		Items:      items,
		Command:    strings.TrimSpace(command),
		Duration:   time.Since(start),
		Success:    err == nil,
		Err:        err,
		ExecutedAt: time.Now(),
	}
	if err != nil {
		result.ErrorCode = ClassifyMikrotikError(err)
	}
	return result
}

func commandFailure(command string, err error, duration time.Duration) *MikrotikCommandResult {
	return &MikrotikCommandResult{
		Command:    strings.TrimSpace(command),
		Duration:   duration,
		Success:    false,
		ErrorCode:  ClassifyMikrotikError(err),
		Err:        err,
		ExecutedAt: time.Now(),
	}
}
