package log

import (
	"github.com/golang/protobuf/ptypes/any"
	"github.com/r-moraru/modular-raft/proto/entries"
)

type Log interface {
	GetLastIndex() uint64
	GetEntry(index uint64) (*entries.LogEntry, error)
	GetTermAtIndex(index uint64) (uint64, error)
	InsertLogEntry(*entries.LogEntry) error
	AppendEntry(term uint64, clientID string, serializationID uint64, entry *any.Any) error
}

// Implementation:
// 		Use read write waitgroup
