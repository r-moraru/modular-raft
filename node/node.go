package node

import (
	"context"
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
	SetCurrentLeaderID(leaderId string)
	SetCommitIndex(commitIndex uint64)
	SetCurrentTerm(newTerm uint64)
	VotedForTerm() bool
	ClearVotedFor()
	ResetTimer()
	GetCurrentLeaderID() string

	Run(ctx context.Context)
}
