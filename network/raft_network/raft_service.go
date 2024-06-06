package raft_network

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/r-moraru/modular-raft/log"
	"github.com/r-moraru/modular-raft/node"
	pb "github.com/r-moraru/modular-raft/proto/raft_service"
)

type RaftService struct {
	pb.UnimplementedRaftServiceServer
	RaftNode node.Node
	Log      log.Log
}

func (r *RaftService) buildAppendEntriesResponse(success bool) *pb.AppendEntriesResponse {
	return &pb.AppendEntriesResponse{Term: r.RaftNode.GetCurrentTerm(), Success: success}
}

func (r *RaftService) buildRequestVoteResponse(voteGranted bool) *pb.RequestVoteResponse {
	return &pb.RequestVoteResponse{Term: r.RaftNode.GetCurrentTerm(), VoteGranted: voteGranted}
}

func (r *RaftService) AppendEntries(ctx context.Context, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	slog.Info(fmt.Sprintf("APPEND ENTRIES - Term from request: %d, Current term: %d\n", req.GetTerm(), r.RaftNode.GetCurrentTerm()))
	if r.RaftNode.GetCurrentTerm() > req.GetTerm() {
		slog.Info("Request term is behind - rejected.")
		return r.buildAppendEntriesResponse(false), nil
	}

	r.RaftNode.SetCurrentLeaderID(req.LeaderId)

	r.RaftNode.ResetTimer()

	if r.RaftNode.GetState() != node.Follower {
		slog.Info("Stepping down, becoming follower.")
		r.RaftNode.SetState(node.Follower)
		r.RaftNode.ClearVotedFor()
		r.RaftNode.SetCurrentTerm(req.GetTerm())
	} else {
		slog.Info("node is already follower for some reason.")
	}

	if req.Entry == nil {
		// Received heartbeat
		return r.buildAppendEntriesResponse(true), nil
	}

	if r.Log.GetLength() < req.GetPrevLogIndex() {
		return r.buildAppendEntriesResponse(false), nil
	}
	termOfPrevLogIndex, err := r.Log.GetTermAtIndex(req.GetPrevLogIndex())
	if err != nil {
		slog.Error(fmt.Sprintf("Append entry - unable to get term at index %d from local log.", req.GetPrevLogIndex()))
		return r.buildAppendEntriesResponse(false), err
	}
	if termOfPrevLogIndex != req.GetPrevLogTerm() {
		return r.buildAppendEntriesResponse(false), nil
	}

	if r.Log.GetLastIndex() >= req.Entry.Index {
		termOfLogIndex, err := r.Log.GetTermAtIndex(req.Entry.Index)
		if err != nil {
			slog.Error(fmt.Sprintf("Append entry - unable to get term at index %d from local log.", req.Entry.Index))
			return r.buildAppendEntriesResponse(false), err
		}
		if termOfLogIndex == req.Entry.Term {
			// already appended
			return r.buildAppendEntriesResponse(true), nil
		}
	}

	if req.GetLeaderCommit() > r.RaftNode.GetCommitIndex() {
		r.RaftNode.SetCommitIndex(min(req.GetLeaderCommit(), req.GetEntry().GetIndex()))
	}

	err = r.Log.InsertLogEntry(req.Entry)
	if err != nil {
		slog.Error("Append entry - failed to append entry.")
		return r.buildAppendEntriesResponse(false), err
	}

	return r.buildAppendEntriesResponse(true), nil
}

func (r *RaftService) RequestVote(ctx context.Context, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	slog.Info(fmt.Sprintf("Term from request: %d, Current term: %d\n", req.GetTerm(), r.RaftNode.GetCurrentTerm()))
	// TODO: step down if needed - become follower
	if req.GetTerm() < r.RaftNode.GetCurrentTerm() {
		return r.buildRequestVoteResponse(false), nil
	}

	if r.RaftNode.VotedForTerm() && r.RaftNode.GetVotedFor() != req.GetCandidateId() {
		slog.Info(fmt.Sprintln("ALREADY VOTED"))
		return r.buildRequestVoteResponse(false), nil
	}

	var lastTerm uint64
	var err error
	lastIndex := r.Log.GetLastIndex()
	if lastIndex == 0 {
		lastTerm = 0
	} else {
		lastTerm, err = r.Log.GetTermAtIndex(lastIndex)
		if err != nil {
			slog.Error(err.Error())
			return r.buildRequestVoteResponse(false), err
		}
	}

	if req.GetLastLogIndex() < lastIndex || req.GetLastLogTerm() < lastTerm {
		slog.Info(fmt.Sprintf(
			"request last log index: %d, local log last index: %d, request last log term: %d, current term: %d\n",
			req.GetLastLogIndex(),
			r.Log.GetLastIndex(),
			req.GetLastLogTerm(),
			r.RaftNode.GetCurrentTerm()))
		slog.Info(fmt.Sprintln("CANDIDATE LOG NOT UP TO DATE"))
		return r.buildRequestVoteResponse(false), nil
	}

	r.RaftNode.ResetTimer()

	slog.Info("Stepping down, becoming follower.")
	r.RaftNode.SetState(node.Follower)
	r.RaftNode.SetCurrentTerm(req.GetTerm())
	r.RaftNode.SetVotedFor(req.GetCandidateId())

	return r.buildRequestVoteResponse(true), nil
}
