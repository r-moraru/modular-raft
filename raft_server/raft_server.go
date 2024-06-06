package raft_server

import (
	"encoding/json"
	logger "log"
	"net/http"

	"github.com/r-moraru/modular-raft/log"
	"github.com/r-moraru/modular-raft/node"
	"github.com/r-moraru/modular-raft/state_machine"
)

type ReplicationRequest struct {
	ClientID        string `json:"client_id"`
	SerializationID uint64 `json:"serialization_id"`
	Entry           string `json:"entry"`
}

type ReplicationResponse struct {
}

type RaftServer struct {
	Log          log.Log
	Node         node.Node
	StateMachine state_machine.StateMachine
}

func (s *RaftServer) CreateReplicationHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		replicationRequest := new(ReplicationRequest)
		err := json.NewDecoder(req.Body).Decode(replicationRequest)
		if err != nil {
			http.Error(w, "Unable to decode replication request.", http.StatusInternalServerError)
			return
		}

		replicationResponse, err := s.HandleReplicationRequest(
			ctx,
			replicationRequest.ClientID,
			replicationRequest.SerializationID,
			replicationRequest.Entry,
		)
		if err != nil {
			http.Error(w, "Internal error retrieving response.", http.StatusInternalServerError)
			return
		}
		res, err := json.Marshal(replicationResponse)
		if err != nil {
			http.Error(w, "Internal error sending response.", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(res)
	}
}

func (s *RaftServer) Run(listenAddr string) {
	http.HandleFunc("/replicate", s.CreateReplicationHandler())
	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		logger.Fatalf("Raft server run failed.")
	}
}
