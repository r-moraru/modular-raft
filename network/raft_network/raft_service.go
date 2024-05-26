package raft_service

import (
	"context"

	"github.com/r-moraru/modular-raft/node"
	pb "github.com/r-moraru/modular-raft/proto/raft_service"
)

type RaftService struct {
	pb.UnimplementedRaftServiceServer
	raftNode *node.Node
}

func (r *RaftService) buildAppendEntriesResponse(success bool) *pb.AppendEntriesResponse {
	return &pb.AppendEntriesResponse{Term: r.raftNode.GetCurrentTerm(), Success: success}
}

func (r *RaftService) buildRequestVoteResponse(voteGranted bool) *pb.RequestVoteResponse {
	return &pb.RequestVoteResponse{Term: r.raftNode.GetCurrentTerm(), VoteGranted: voteGranted}
}

func (r *RaftService) AppendEntries(ctx context.Context, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	if r.raftNode.GetCurrentTerm() > req.GetTerm() {
		return r.buildAppendEntriesResponse(false), nil
	}
	if r.raftNode.GetLogLength() <= req.GetPrevLogIndex() {
		return r.buildAppendEntriesResponse(false), nil
	}
	termOfPrevLogIndex, err := r.raftNode.GetTermAtIndex(req.GetPrevLogIndex())
	if err != nil {
		// TODO: log errors
		return r.buildAppendEntriesResponse(false), err
	}
	if termOfPrevLogIndex != req.GetPrevLogTerm() {
		return r.buildAppendEntriesResponse(false), nil
	}

	err = r.raftNode.AppendEntry(req.Entry)
	if err != nil {
		return r.buildAppendEntriesResponse(false), err
	}

	return r.buildAppendEntriesResponse(true), nil
}

func (r *RaftService) RequestVote(ctx context.Context, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	if req.GetTerm() < r.raftNode.GetCurrentTerm() {
		return r.buildRequestVoteResponse(false), nil
	}

	if r.raftNode.VotedForTerm(req.GetTerm()) && string(r.raftNode.GetVotedFor()) != req.GetCandidateId() {
		return r.buildRequestVoteResponse(false), nil
	}

	if req.GetLastLogIndex() < r.raftNode.GetLastLogIndex() || req.GetLastLogTerm() < r.raftNode.GetCurrentTerm() {
		return r.buildRequestVoteResponse(false), nil
	}

	r.raftNode.SetCurrentTerm(req.GetTerm())
	r.raftNode.SetVotedForTerm(req.GetTerm(), true)
	r.raftNode.SetVotedFor(req.GetCandidateId())

	// TODO: update raft node term and set voted for
	return r.buildRequestVoteResponse(true), nil
}
