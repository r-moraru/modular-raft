package kv_store

import (
	"errors"
	"sync"

	"github.com/r-moraru/modular-raft/proto/entries"
)

var (
	LogIndexOutOfBoundsError = errors.New("log index out of bounds")
)

type InMemoryLog struct {
	entries []*entries.LogEntry
	Mutex   *sync.Mutex
}

func (l *InMemoryLog) GetLastIndex() uint64 {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()
	if len(l.entries) == 0 {
		return 0
	}
	return l.entries[len(l.entries)-1].Index
}

func (l *InMemoryLog) GetLength() uint64 {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()
	return uint64(len(l.entries))
}

func (l *InMemoryLog) GetTermAtIndex(index uint64) (uint64, error) {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()
	if uint64(len(l.entries)) <= index-1 {
		return 0, LogIndexOutOfBoundsError
	}
	return l.entries[index-1].Term, nil
}

func (l *InMemoryLog) GetEntry(index uint64) (*entries.LogEntry, error) {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()
	if uint64(len(l.entries)) <= index-1 {
		return nil, LogIndexOutOfBoundsError
	}
	return l.entries[index-1], nil
}

func (l *InMemoryLog) InsertLogEntry(entry *entries.LogEntry) error {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()
	if uint64(len(l.entries)) < entry.Index-1 {
		return LogIndexOutOfBoundsError
	}
	l.entries = l.entries[:entry.Index-1]
	l.entries = append(l.entries, entry)
	return nil
}

func (l *InMemoryLog) AppendEntry(term uint64, clientID string, serializationID uint64, entry string) error {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()
	logEntry := &entries.LogEntry{
		Index:           uint64(len(l.entries) + 1),
		Term:            term,
		ClientID:        clientID,
		SerializationID: serializationID,
		Entry:           entry,
	}
	l.entries = append(l.entries, logEntry)
	return nil
}
