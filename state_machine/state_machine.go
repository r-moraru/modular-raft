package state_machine

import (
	"github.com/golang/protobuf/ptypes/any"
)

type StateMachine interface {
	Apply(*any.Any) error
}
