package kv_store

import (
	"context"
	"fmt"
	"strconv"

	"github.com/r-moraru/modular-raft/proto/entries"
	"github.com/r-moraru/modular-raft/state_machine"
	"golang.org/x/exp/slog"
)

type KvStore struct {
	lastApplied uint64
}

func (k *KvStore) Apply(entry *entries.LogEntry) error {
	slog.Info(fmt.Sprintf(
		"Applying entry with index %d, term %d, clientId %s, serializationId %d.\n",
		entry.Index,
		entry.Term,
		entry.ClientID,
		entry.SerializationID,
	))
	k.lastApplied = entry.Index
	return nil
}

func (k *KvStore) GetLastApplied() uint64 {
	return k.lastApplied
}

func (k *KvStore) WaitForResult(_ context.Context, clientID string, serializationID uint64) chan state_machine.ApplyResult {
	slog.Info("Waiting for result of request with clientID " + clientID + ", serializationID " + strconv.Itoa(int(serializationID)) + ".\n")
	resChan := make(chan state_machine.ApplyResult, 1)
	resChan <- state_machine.ApplyResult{
		Result: "success",
		Error:  nil,
	}
	return resChan
}
