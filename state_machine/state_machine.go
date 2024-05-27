package state_machine

import (
	"context"

	"github.com/golang/protobuf/ptypes/any"
)

type ApplyResult struct {
	Result *any.Any
	Error  error
}

type StateMachine interface {
	Apply(*any.Any) error
	GetLastApplied() uint64
	WaitForResult(ctx context.Context, clientID string, serializationID uint64) chan ApplyResult
}
