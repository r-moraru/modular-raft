package state_machine

import (
	"context"

	"github.com/golang/protobuf/ptypes/any"
	"github.com/r-moraru/modular-raft/proto/entries"
)

type ApplyResult struct {
	Result *any.Any
	Error  error
}

type StateMachine interface {
	Apply(*entries.LogEntry) error
	GetLastApplied() uint64
	WaitForResult(ctx context.Context, clientID string, serializationID uint64) chan ApplyResult
}
