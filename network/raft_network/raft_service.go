package raft_service

import (
	"context"
	logger "log"

	"github.com/r-moraru/modular-raft/log"
	"github.com/r-moraru/modular-raft/node"
	pb "github.com/r-moraru/modular-raft/proto/raft_service"
)

type RaftService struct {
	pb.UnimplementedRaftServiceServer
	raftNode node.Node
	log      log.Log
}

func (r *RaftService) buildAppendEntriesResponse(success bool) *pb.AppendEntriesResponse {
	return &pb.AppendEntriesResponse{Term: r.raftNode.GetCurrentTerm(), Success: success}
}

func (r *RaftService) buildRequestVoteResponse(voteGranted bool) *pb.RequestVoteResponse {
	return &pb.RequestVoteResponse{Term: r.raftNode.GetCurrentTerm(), VoteGranted: voteGranted}
}

func (r *RaftService) AppendEntries(ctx context.Context, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	if r.raftNode.GetCurrentTerm() >= req.GetTerm() {
		return r.buildAppendEntriesResponse(false), nil
	}

	r.raftNode.SetCurrentLeaderId(req.LeaderId)

	if r.raftNode.GetState() != node.Follower {
		r.raftNode.SetState(node.Follower)
		r.raftNode.ClearVotedFor()
		r.raftNode.SetCurrentTerm(req.GetTerm())
	}

	r.raftNode.ResetTimer()

	if r.log.GetLength() <= req.GetPrevLogIndex() {
		return r.buildAppendEntriesResponse(false), nil
	}
	termOfPrevLogIndex, err := r.log.GetTermAtIndex(req.GetPrevLogIndex())
	if err != nil {
		logger.Fatalf("Append entry - unable to get term at index %d from local log.", req.GetPrevLogIndex())
		return r.buildAppendEntriesResponse(false), err
	}
	if termOfPrevLogIndex != req.GetPrevLogTerm() {
		return r.buildAppendEntriesResponse(false), nil
	}

	if req.Entry == nil {
		// Received heartbeat
		return r.buildAppendEntriesResponse(true), nil
	}

	if r.log.GetLastIndex() >= req.Entry.Index {
		termOfLogIndex, err := r.log.GetTermAtIndex(req.Entry.Index)
		if err != nil {
			logger.Fatalf("Append entry - unable to get term at index %d from local log.", req.Entry.Index)
			return r.buildAppendEntriesResponse(false), err
		}
		if termOfLogIndex == req.Entry.Term {
			// already appended
			return r.buildAppendEntriesResponse(true), nil
		}
	}

	if req.GetLeaderCommit() > r.raftNode.GetCommitIndex() {
		r.raftNode.SetCommitIndex(min(req.GetLeaderCommit(), req.GetEntry().GetIndex()))
	}

	err = r.log.InsertLogEntry(req.Entry)
	if err != nil {
		logger.Fatal("Append entry - failed to append entry.")
		return r.buildAppendEntriesResponse(false), err
	}

	return r.buildAppendEntriesResponse(true), nil
}

func (r *RaftService) RequestVote(ctx context.Context, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	if req.GetTerm() <= r.raftNode.GetCurrentTerm() {
		return r.buildRequestVoteResponse(false), nil
	}

	if r.raftNode.VotedForTerm() && string(r.raftNode.GetVotedFor()) != req.GetCandidateId() {
		return r.buildRequestVoteResponse(false), nil
	}

	if req.GetLastLogIndex() < r.log.GetLastIndex() || req.GetLastLogTerm() < r.raftNode.GetCurrentTerm() {
		return r.buildRequestVoteResponse(false), nil
	}

	r.raftNode.SetCurrentTerm(req.GetTerm())
	r.raftNode.SetVotedFor(req.GetCandidateId())

	return r.buildRequestVoteResponse(true), nil
}
