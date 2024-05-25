package log

import (
	"github.com/golang/protobuf/ptypes/any"
	"github.com/r-moraru/modular-raft/proto/entries"
)

type Log interface {
	GetEntry(index int64) *entries.LogEntry
	GetTermAtIndex(index int64) int64
	InsertLogEntry(*entries.LogEntry)
	AppendEntry(term int64, entry *any.Any)
}

// Implementation:
// 		Use read write waitgroup
