package network

import (
	"github.com/r-moraru/modular-raft/proto/entries"
)

type PeerId string

type ResponseStatus int64

const (
	Success ResponseStatus = iota
	LogInconsistency
	NotReceived
)

type Network interface {
	GetId() PeerId
	CheckGreatestTerm() int64
	CheckUpdateTimer() bool
	GetNewLogEntry() *entries.LogEntry
	GetUpdatedCommitIndex(previousCommitIndex int64) int64

	SendRequestVoteAsync(term int64)
	SendHeartbeatAsync(peerId PeerId)
	SendAppendEntryAsync(peerId PeerId, logEntry *entries.LogEntry)

	GotMajorityVote() chan bool
	GetAppendEntryResponse(index, term int64, peerId PeerId) ResponseStatus
}
