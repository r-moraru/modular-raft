package state_machine

import (
	"context"

	"github.com/r-moraru/modular-raft/proto/entries"
)

type ApplyResult struct {
	Result string
	Error  error
}

type StateMachine interface {
	Apply(*entries.LogEntry) error
	GetLastApplied() uint64 // TODO: add error
	WaitForResult(ctx context.Context, clientID string, serializationID uint64) chan ApplyResult
}
