package network

import (
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
	CheckGreatestTerm() uint64
	CheckUpdateTimer() bool
	GetNewLogEntry() *entries.LogEntry
	GetUpdatedCommitIndex(previousCommitIndex uint64) uint64

	SendRequestVoteAsync(term uint64)
	SendHeartbeatAsync(peerId string)
	SendAppendEntryAsync(peerId string, logEntry *entries.LogEntry)

	GotMajorityVote() chan bool
	GetAppendEntryResponse(index, term uint64, peerId string) ResponseStatus
}
