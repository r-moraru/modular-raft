package raft_server

import (
	"context"
	"encoding/json"
	"fmt"
	logger "log"
	"log/slog"
	"net"
	"net/http"

	"github.com/r-moraru/modular-raft/log"
	"github.com/r-moraru/modular-raft/network/raft_network"
	"github.com/r-moraru/modular-raft/node"
	"github.com/r-moraru/modular-raft/node/raft_node"
	"github.com/r-moraru/modular-raft/proto/raft_service"
	"github.com/r-moraru/modular-raft/state_machine"
	"google.golang.org/grpc"
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
	network      *raft_network.Network
}

func New(log log.Log, stateMachine state_machine.StateMachine, electionTimeout uint64, heartbeat uint64) (*RaftServer, error) {
	network := &raft_network.Network{}
	node, err := raft_node.New(electionTimeout, heartbeat, log, stateMachine, network)
	if err != nil {
		return nil, err
	}
	return &RaftServer{
		Log:          log,
		StateMachine: stateMachine,
		Node:         node,
		network:      network,
	}, nil
}

func (s *RaftServer) Start(nodeNum int, nodeIds []string, nodeAddrs []string) {
	slog.Info("Starting raft consensus server at " + nodeAddrs[nodeNum])
	lis, err := net.Listen("tcp", nodeAddrs[nodeNum])
	if err != nil {
		fmt.Println("failed" + err.Error())
		return
	}

	raftService := &raft_network.RaftService{
		RaftNode: s.Node,
		Log:      s.Log,
	}
	server := grpc.NewServer()
	raft_service.RegisterRaftServiceServer(server, raftService)
	go func() {
		if err := server.Serve(lis); err != nil {
			fmt.Println("failed serve")
		}
	}()

	s.network.Init(s.Node, s.Log, nodeNum, nodeIds, nodeAddrs)

	go func() {
		s.Node.Run(context.Background())
	}()
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
