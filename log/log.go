package log

import (
	"github.com/golang/protobuf/ptypes/any"
	"github.com/r-moraru/modular-raft/proto/entries"
)

type Log interface {
	GetLastIndex() int64
	GetEntry(index int64) (*entries.LogEntry, error)
	GetTermAtIndex(index int64) (int64, error)
	InsertLogEntry(*entries.LogEntry) error
	AppendEntry(term int64, clientID string, serializationID int64, entry *any.Any) error
}

// Implementation:
// 		Use read write waitgroup
