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
	GetPeerList() []string

	SendRequestVote(context context.Context, term uint64) chan bool
	SendHeartbeat(peerId string)
	SendAppendEntry(ctx context.Context, peerId string, logEntry *entries.LogEntry) ResponseStatus
}
