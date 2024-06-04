package node

import (
	"context"

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
	ReplicationStatus ReplicationStatus `json:"replication_status"`
	LeaderID          string            `json:"leader_id"`
	Result            string            `json:"result"`
}

type Node interface {
	GetState() State
	SetState(newState State)
	GetVotedFor() string
	SetVotedFor(peerId string)
	GetCurrentTerm() uint64
	GetCommitIndex() uint64
	SetCurrentLeaderId(leaderId string)
	SetCommitIndex(commitIndex uint64)
	SetCurrentTerm(newTerm uint64)
	SetVotedForTerm(term uint64, voted bool)
	VotedForTerm() bool
	ClearVotedFor()
	ResetTimer()
	GetCurrentLeaderID() string

	Run(ctx context.Context)

	GetLogLength() uint64
	GetLastLogIndex() uint64
	GetTermAtIndex(index uint64) (uint64, error)
	AppendEntry(entry *entries.LogEntry) error
}
