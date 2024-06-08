package network

import (
	"context"

	"github.com/r-moraru/modular-raft/proto/entries"
)

type ResponseStatus uint64

const (
	Success ResponseStatus = iota
	LogInconsistency
	NotReceived
	TermIssue
)

type Network interface {
	GetId() string
	GetPeerList() []string

	SendRequestVote(ctx context.Context, term uint64) chan bool
	SendHeartbeat(ctx context.Context, peerId string, prevIndex uint64) ResponseStatus
	SendAppendEntry(ctx context.Context, peerId string, logEntry *entries.LogEntry) ResponseStatus
}
