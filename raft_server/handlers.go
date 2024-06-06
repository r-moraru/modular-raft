package raft_server

import (
	"context"
	logger "log"

	"github.com/r-moraru/modular-raft/node"
)

func (s *RaftServer) HandleReplicationRequest(ctx context.Context, clientID string, serializationID uint64, entry string) (node.ReplicationResponse, error) {
	res := node.ReplicationResponse{}
	if s.Node.GetState() != node.Leader {
		res.ReplicationStatus = node.NotLeader
		// best effort, might not be leader
		res.LeaderID = s.Node.GetCurrentLeaderID()
		return res, nil
	}

	s.Log.AppendEntry(s.Node.GetCurrentTerm(), clientID, serializationID, entry)

	select {
	case <-ctx.Done():
		return res, nil
	case result := <-s.StateMachine.WaitForResult(ctx, clientID, serializationID):
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
