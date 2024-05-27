package raft_server

import (
	"context"
	logger "log"

	"github.com/r-moraru/modular-raft/log"
	"github.com/r-moraru/modular-raft/node"
	"github.com/r-moraru/modular-raft/state_machine"
	"google.golang.org/protobuf/types/known/anypb"
)

type RaftServer struct {
	log          log.Log
	node         node.Node
	stateMachine state_machine.StateMachine
}

func (s *RaftServer) HandleReplicationRequest(ctx context.Context, clientID string, serializationID uint64, entry *anypb.Any) (node.ReplicationResponse, error) {
	res := node.ReplicationResponse{}
	if s.node.GetState() != node.Leader {
		res.ReplicationStatus = node.NotLeader
		// best effort, might not be leader
		res.LeaderID = s.node.GetCurrentLeaderID()
		return res, nil
	}

	s.log.AppendEntry(s.node.GetCurrentTerm(), clientID, serializationID, entry)

	select {
	case <-ctx.Done():
		return res, nil
	case result := <-s.stateMachine.WaitForResult(ctx, clientID, serializationID):
		if result.Error != nil {
			logger.Fatalf("State machine returned error for clientID %s, serializationID %d.", clientID, serializationID)
			res.ReplicationStatus = node.ApplyError
			return res, nil
		}
		res.Result = result.Result
		res.ReplicationStatus = node.Replicated
		return res, nil
	}
}

func (n *RaftServer) HandleQueryRequest(ctx context.Context) {
	// TODO: implement query request handler
}
