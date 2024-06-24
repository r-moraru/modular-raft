package log

import (
	"github.com/r-moraru/modular-raft/proto/entries"
)

type Log interface {
	GetLastIndex() uint64 // TODO: add error, remove get length
	GetLength() uint64
	GetEntry(index uint64) (*entries.LogEntry, error)
	GetTermAtIndex(index uint64) (uint64, error)
	InsertLogEntry(*entries.LogEntry) error
	AppendEntry(term uint64, clientID string, serializationID uint64, entry string) error
}

func GetTermAtIndexHelper(l Log, index uint64) (uint64, error) {
	if index == 0 {
		return 0, nil
	}
	return l.GetTermAtIndex(index)
}
