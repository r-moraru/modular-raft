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
	WaitForResult(ctx context.Context, clientID string, serializationID int64) chan ApplyResult
}
