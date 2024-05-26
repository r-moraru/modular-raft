package raft_service

import (
	"context"
	"log"

	"github.com/r-moraru/modular-raft/node"
	pb "github.com/r-moraru/modular-raft/proto/raft_service"
)

type RaftService struct {
	pb.UnimplementedRaftServiceServer
	raftNode node.Node
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

	if r.raftNode.GetState() != node.Follower {
		r.raftNode.SetState(node.Follower)
		r.raftNode.SetCurrentTerm(req.GetTerm())
	}

	r.raftNode.ResetTimer()

	if r.raftNode.GetLogLength() <= req.GetPrevLogIndex() {
		return r.buildAppendEntriesResponse(false), nil
	}
	termOfPrevLogIndex, err := r.raftNode.GetTermAtIndex(req.GetPrevLogIndex())
	if err != nil {
		log.Fatalf("Append entry - unable to get term at index %d from local log.", req.GetPrevLogIndex())
		return r.buildAppendEntriesResponse(false), err
	}
	if termOfPrevLogIndex != req.GetPrevLogTerm() {
		return r.buildAppendEntriesResponse(false), nil
	}

	if req.Entry == nil {
		// Received heartbeat
		return r.buildAppendEntriesResponse(true), nil
	}

	if r.raftNode.GetLastLogIndex() >= req.Entry.Index {
		termOfLogIndex, err := r.raftNode.GetTermAtIndex(req.Entry.Index)
		if err != nil {
			log.Fatalf("Append entry - unable to get term at index %d from local log.", req.Entry.Index)
			return r.buildAppendEntriesResponse(false), err
		}
		if termOfLogIndex == req.Entry.Term {
			// already appended
			return r.buildAppendEntriesResponse(true), nil
		}
	}

	err = r.raftNode.AppendEntry(req.Entry)
	if err != nil {
		log.Fatal("Append entry - failed to append entry.")
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

	return r.buildRequestVoteResponse(true), nil
}
