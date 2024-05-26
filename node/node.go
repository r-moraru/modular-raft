package node

import (
	"context"

	"github.com/golang/protobuf/ptypes/any"
	"github.com/r-moraru/modular-raft/proto/entries"
)

type State uint64

const (
	Leader State = iota
	Follower
	Candidate
)

type ReplicationStatus uint64

const (
	NotLeader ReplicationStatus = iota
	NotTracked
	InProgress
	Replicated
	ApplyError
)

type ReplicationResponse struct {
	ReplicationStatus ReplicationStatus
	LeaderID          string
	Result            *any.Any
}

type Node interface {
	GetState() State
	SetState(newState State)
	GetVotedFor() string
	SetVotedFor(peerId string)
	GetCurrentTerm() uint64
	SetCurrentTerm(newTerm uint64)
	SetVotedForTerm(term uint64, voted bool)
	VotedForTerm(term uint64) bool
	ResetTimer()

	Run(ctx context.Context)

	HandleReplicationRequest(ctx context.Context, clientID string, serializationID uint64, entry *any.Any) (ReplicationResponse, error)
	HandleQueryRequest(ctx context.Context)

	GetLogLength() uint64
	GetLastLogIndex() uint64
	GetTermAtIndex(index uint64) (uint64, error)
	AppendEntry(entry *entries.LogEntry) error
}
