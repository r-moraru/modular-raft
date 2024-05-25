package network

import (
	"github.com/golang/protobuf/ptypes/any"
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
	CheckGreatestTerm() int64
	CheckUpdateTimer() bool
	UpdateCommitIndex(previousCommitIndex int64) int64
	GetNewLogEntry() *entries.LogEntry
	GetId() PeerId
	SendRequestVoteAsync(term int64)
	GotMajorityVote() chan bool
	GetRequest() *any.Any
	SendHeartbeatAsync(peerId PeerId)
	SendAppendEntryAsync(peerId PeerId, logEntry *entries.LogEntry)
	GetPeerResponse(index, term int64, peerId PeerId) ResponseStatus
}
