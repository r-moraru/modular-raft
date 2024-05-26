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
)

type Network interface {
	GetId() string

	SendRequestVoteAsync(term uint64)
	SendHeartbeat(peerId string)
	SendAppendEntry(ctx context.Context, peerId string, logEntry *entries.LogEntry) ResponseStatus

	GotMajorityVote(ctx context.Context) chan bool
}
