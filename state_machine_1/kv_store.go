package state_machine_1

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/r-moraru/modular-raft/clients"
	"github.com/r-moraru/modular-raft/proto/entries"
	"github.com/r-moraru/modular-raft/state_machine"
)

type applicationStatus int

type applicationEntry struct {
	status applicationStatus
	result string
}

const (
	inProgress applicationStatus = iota
	done
	failed
	tracked
)

var (
	ErrEntryTracked    = errors.New("entry already tracked")
	ErrEntryNotTracked = errors.New("entry not tracked")
	ErrTimedOut        = errors.New("timed out before getting result")
	ErrInvalidIndex    = errors.New("index is invalid, must be lastApplied + 1")
	ErrBadRequest      = errors.New("bad request format")
)

type KvStoreServer struct {
	// Metadata
	lastApplied      uint64
	lastAppliedMutex *sync.RWMutex
	status           map[string]map[uint64]applicationEntry
	statusLock       *sync.RWMutex

	store     map[string]string
	storeLock *sync.RWMutex
}

type request struct {
	RequestType string `json:"request_type"`
	Key         string `json:"key"`
	Value       string `json:"value"`
}

type response struct {
	Success  bool   `json:"success,omitempty"`
	KeyFound bool   `json:"key_found,omitempty"`
	Value    string `json:"value,omitempty"`
}

func New() *KvStoreServer {
	return &KvStoreServer{
		store:            make(map[string]string),
		storeLock:        &sync.RWMutex{},
		lastAppliedMutex: &sync.RWMutex{},
		statusLock:       &sync.RWMutex{},
		status:           make(map[string]map[uint64]applicationEntry),
	}
}

func (s *KvStoreServer) Run(ctx context.Context, listenAddr string) {
	http.HandleFunc(clients.ApplyPath, s.createApplyHandler())
	http.HandleFunc(clients.LastAppliedPath, s.createLastAppliedHandler())
	http.HandleFunc(clients.GetResultPath, s.CreateWaitForResultHandler())

	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Fatalf(err.Error())
	}
}

func (s *KvStoreServer) getLastApplied() uint64 {
	s.lastAppliedMutex.RLock()
	defer s.lastAppliedMutex.RUnlock()
	return s.lastApplied
}

func (s *KvStoreServer) incrementLastApplied() {
	s.lastAppliedMutex.Lock()
	defer s.lastAppliedMutex.Unlock()
	s.lastApplied += 1
}

func checkRequest(req *request) bool {
	switch req.RequestType {
	case "write":
		if req.Key == "" {
			return false
		}
	case "read":
		if req.Key == "" {
			return false
		}
	case "delete":
		if req.Key == "" {
			return false
		}
	default:
		return false
	}

	return true
}

func (s *KvStoreServer) write(key, value string) {
	s.storeLock.Lock()
	defer s.storeLock.Unlock()
	s.store[key] = value
}

func (s *KvStoreServer) get(key string) (string, bool) {
	s.storeLock.RLock()
	defer s.storeLock.RUnlock()
	val, found := s.store[key]
	if !found {
		return "", false
	}
	return val, true
}

func (s *KvStoreServer) delete(key string) bool {
	s.storeLock.Lock()
	defer s.storeLock.Unlock()
	_, found := s.store[key]
	if !found {
		return false
	}
	delete(s.store, key)
	return true
}

func (s *KvStoreServer) processRequest(req *request) response {
	resp := response{}
	switch req.RequestType {
	case "write":
		s.write(req.Key, req.Value)
		resp.Success = true
	case "read":
		val, found := s.get(req.Key)
		resp.Value = val
		resp.KeyFound = found
	case "delete":
		found := s.delete(req.Key)
		resp.KeyFound = found
	}
	return resp
}

func (s *KvStoreServer) apply(entry *entries.LogEntry) error {

	if entry.Index != s.getLastApplied()+1 {
		return ErrInvalidIndex
	}

	clientEntry, found := s.status[entry.ClientID]
	if !found {
		s.status[entry.ClientID] = make(map[uint64]applicationEntry)
	}
	_, found = clientEntry[entry.SerializationID]
	if !found {
		req := new(request)
		err := json.Unmarshal([]byte(entry.Entry), req)
		if err != nil || !checkRequest(req) {
			return ErrBadRequest
		}

		s.setApplicationEntry(entry.ClientID, entry.SerializationID, applicationEntry{
			status: inProgress,
		})

		go func() {
			result := s.processRequest(req)
			resp, _ := json.Marshal(result)

			s.setApplicationEntry(entry.ClientID, entry.SerializationID, applicationEntry{
				result: string(resp),
				status: done,
			})
			s.incrementLastApplied()
		}()

		return nil
	}
	return ErrEntryTracked
}

func (s *KvStoreServer) createLastAppliedHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		lastAppliedResponse := clients.GetLastAppliedResponse{
			LastApplied: s.getLastApplied(),
		}

		res, err := json.Marshal(lastAppliedResponse)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(res)
	}
}

func (s *KvStoreServer) createApplyHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		entry := new(entries.LogEntry)
		err := json.NewDecoder(req.Body).Decode(entry)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = s.apply(entry)
		if err != nil && err != ErrEntryTracked {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (s *KvStoreServer) getApplicationEntry(clientID string, serializationID uint64) *applicationEntry {
	s.statusLock.RLock()
	defer s.statusLock.RUnlock()

	_, found := s.status[clientID]
	if !found {
		return nil
	}
	appEntry, found := s.status[clientID][serializationID]
	if !found {
		return nil
	}

	return &appEntry
}

func (s *KvStoreServer) setApplicationEntry(clientID string, serializationID uint64, appEntry applicationEntry) {
	s.statusLock.Lock()
	defer s.statusLock.Unlock()

	_, found := s.status[clientID]
	if !found {
		s.status[clientID] = make(map[uint64]applicationEntry)
	}
	s.status[clientID][serializationID] = appEntry
}

func (s *KvStoreServer) waitForResult(ctx context.Context, clientID string, serializationID uint64) chan state_machine.ApplyResult {
	resChan := make(chan state_machine.ApplyResult, 1)

	go func() {
		for {
			select {
			case <-ctx.Done():
				resChan <- state_machine.ApplyResult{Error: ErrTimedOut}
				return
			default:
				appEntry := s.getApplicationEntry(clientID, serializationID)
				if appEntry == nil {
					continue
				}

				if appEntry.status == done {
					resChan <- state_machine.ApplyResult{
						Result: appEntry.result,
					}
				}
			}
		}
	}()

	return resChan
}

func (s *KvStoreServer) CreateWaitForResultHandler() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		getResultRequest := new(clients.GetResultRequest)
		err := json.NewDecoder(req.Body).Decode(getResultRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resChan := s.waitForResult(ctx, getResultRequest.ClientID, getResultRequest.SerializationID)

		select {
		case <-ctx.Done():
			http.Error(w, ErrTimedOut.Error(), http.StatusRequestTimeout)
			return
		case res := <-resChan:
			if res.Error != nil {
				http.Error(w, res.Error.Error(), http.StatusInternalServerError)
				return
			}

			applyResponse := new(clients.GetApplyResultResponse)
			applyResponse.ApplyResult = res.Result

			response, err := json.Marshal(applyResponse)
			if err != nil {
				http.Error(w, err.Error(), http.StatusRequestTimeout)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			w.Write(response)
		}
	}
}
