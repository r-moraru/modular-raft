package raft_network

import (
	"context"
	"testing"
	"time"

	log_mocks "github.com/r-moraru/modular-raft/log/mocks"
	"github.com/r-moraru/modular-raft/network"
	node_mocks "github.com/r-moraru/modular-raft/node/mocks"
	"github.com/r-moraru/modular-raft/proto/entries"
	"github.com/r-moraru/modular-raft/proto/raft_service"
	raft_service_mocks "github.com/r-moraru/modular-raft/proto/raft_service/mocks"
	state_machine_mocks "github.com/r-moraru/modular-raft/state_machine/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type RaftNetworkTestSuite struct {
	suite.Suite

	network *Network

	lastIndex    uint64
	lastTerm     uint64
	currentTerm  uint64
	commitIndex  uint64
	peerMocks    map[string]*raft_service_mocks.RaftServiceClient
	log          *log_mocks.Log
	node         *node_mocks.Node
	stateMachine *state_machine_mocks.StateMachine
}

func (s *RaftNetworkTestSuite) SetupTest() {
	s.log = log_mocks.NewLog(s.T())
	s.stateMachine = state_machine_mocks.NewStateMachine(s.T())
	s.node = node_mocks.NewNode(s.T())
	s.peerMocks = map[string]*raft_service_mocks.RaftServiceClient{
		"node2": raft_service_mocks.NewRaftServiceClient(s.T()),
		"node3": raft_service_mocks.NewRaftServiceClient(s.T()),
		"node4": raft_service_mocks.NewRaftServiceClient(s.T()),
		"node5": raft_service_mocks.NewRaftServiceClient(s.T()),
	}
	s.network = &Network{
		nodeId:          "node1",
		electionTimeout: 50,
		peers: map[string]raft_service.RaftServiceClient{
			"node2": s.peerMocks["node2"],
			"node3": s.peerMocks["node3"],
			"node4": s.peerMocks["node4"],
			"node5": s.peerMocks["node5"],
		},
		log:  s.log,
		node: s.node,
	}
	s.lastIndex = 13
	s.lastTerm = 3
	s.currentTerm = 3
	s.commitIndex = 13
}

func (s *RaftNetworkTestSuite) mockSuccessfulRequestVoteCalls(peerId string, term uint64) {
	s.log.EXPECT().GetLastIndex().Return(s.lastIndex).Once()
	s.log.EXPECT().GetTermAtIndex(s.lastIndex).Return(s.lastTerm, nil).Once()
	s.peerMocks[peerId].EXPECT().RequestVote(mock.Anything, &raft_service.RequestVoteRequest{
		Term:         term,
		CandidateId:  s.network.GetId(),
		LastLogIndex: s.lastIndex,
		LastLogTerm:  s.lastTerm,
	}).Return(&raft_service.RequestVoteResponse{
		Term:        s.lastTerm - 1,
		VoteGranted: true,
	}, nil)
}

func (s *RaftNetworkTestSuite) mockTermIssueRequestVoteCalls(peerId string, term uint64) {
	s.log.EXPECT().GetLastIndex().Return(s.lastIndex).Once()
	s.log.EXPECT().GetTermAtIndex(s.lastIndex).Return(s.lastTerm, nil).Once()
	s.peerMocks[peerId].EXPECT().RequestVote(mock.Anything, &raft_service.RequestVoteRequest{
		Term:         term,
		CandidateId:  s.network.GetId(),
		LastLogIndex: s.lastIndex,
		LastLogTerm:  s.lastTerm,
	}).Return(&raft_service.RequestVoteResponse{
		Term:        s.lastTerm + 1,
		VoteGranted: false,
	}, nil)
}

func (s *RaftNetworkTestSuite) mockTimedOutRequestVoteCalls(peerId string, term uint64) {
	s.log.EXPECT().GetLastIndex().Return(s.lastIndex).Once()
	s.log.EXPECT().GetTermAtIndex(s.lastIndex).Return(s.lastTerm, nil).Once()
	timer := time.NewTimer(time.Duration(s.network.electionTimeout*2) * time.Millisecond)
	s.peerMocks[peerId].EXPECT().RequestVote(mock.Anything, &raft_service.RequestVoteRequest{
		Term:         term,
		CandidateId:  s.network.GetId(),
		LastLogIndex: s.lastIndex,
		LastLogTerm:  s.lastTerm,
	}).Return(&raft_service.RequestVoteResponse{
		Term:        s.lastTerm + 1,
		VoteGranted: false,
	}, nil).WaitUntil(timer.C)
}

// TODO: test rpc error results
func (s *RaftNetworkTestSuite) TestSendRequestVoteSuccess() {
	t := s.T()

	s.mockSuccessfulRequestVoteCalls("node2", s.lastTerm+1)
	s.mockSuccessfulRequestVoteCalls("node3", s.lastTerm+1)
	s.mockSuccessfulRequestVoteCalls("node4", s.lastTerm+1)
	s.mockSuccessfulRequestVoteCalls("node5", s.lastTerm+1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	majorityVoteChan := s.network.SendRequestVote(ctx, s.lastTerm+1)

	blocked := false
	var gotVoted bool
	select {
	case gotVoted = <-majorityVoteChan:
		blocked = false
	case <-time.After(time.Duration(s.network.electionTimeout) * time.Millisecond):
		blocked = true
	}

	assert.False(t, blocked)
	assert.True(t, gotVoted)
}

func (s *RaftNetworkTestSuite) TestSendRequestVoteFailure() {
	t := s.T()

	s.mockTermIssueRequestVoteCalls("node2", s.lastTerm+1)
	s.mockTermIssueRequestVoteCalls("node3", s.lastTerm+1)
	s.mockTermIssueRequestVoteCalls("node4", s.lastTerm+1)
	s.mockTermIssueRequestVoteCalls("node5", s.lastTerm+1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	majorityVoteChan := s.network.SendRequestVote(ctx, s.lastTerm+1)

	blocked := false
	var gotVoted bool
	select {
	case gotVoted = <-majorityVoteChan:
		blocked = false
	case <-time.After(time.Duration(s.network.electionTimeout) * time.Millisecond):
		blocked = true
	}

	assert.False(t, blocked)
	assert.False(t, gotVoted)
}

func (s *RaftNetworkTestSuite) TestSendRequestVoteVotedByHalf() {
	t := s.T()

	s.mockSuccessfulRequestVoteCalls("node2", s.lastTerm+1)
	s.mockSuccessfulRequestVoteCalls("node3", s.lastTerm+1)
	s.mockTermIssueRequestVoteCalls("node4", s.lastTerm+1)
	s.mockTermIssueRequestVoteCalls("node5", s.lastTerm+1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	majorityVoteChan := s.network.SendRequestVote(ctx, s.lastTerm+1)

	blocked := false
	var gotVoted bool
	select {
	case gotVoted = <-majorityVoteChan:
		blocked = false
	case <-time.After(time.Duration(s.network.electionTimeout) * time.Millisecond):
		blocked = true
	}

	assert.False(t, blocked)
	assert.True(t, gotVoted)
}

func (s *RaftNetworkTestSuite) TestSendRequestVoteTimeout() {
	t := s.T()

	s.mockTimedOutRequestVoteCalls("node2", s.lastTerm+1)
	s.mockTimedOutRequestVoteCalls("node3", s.lastTerm+1)
	s.mockTimedOutRequestVoteCalls("node4", s.lastTerm+1)
	s.mockTimedOutRequestVoteCalls("node5", s.lastTerm+1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	majorityVoteChan := s.network.SendRequestVote(ctx, s.lastTerm+1)

	blocked := false
	var gotVoted bool
	select {
	case gotVoted = <-majorityVoteChan:
		blocked = false
	case <-time.After(time.Duration(s.network.electionTimeout) * time.Millisecond):
		blocked = true
	}

	assert.True(t, blocked)
	assert.False(t, gotVoted)
}

func (s *RaftNetworkTestSuite) TestSendHeartbeatSuccess() {
	s.log.EXPECT().GetLastIndex().Return(s.lastIndex).Once()
	s.log.EXPECT().GetTermAtIndex(s.lastIndex).Return(s.lastTerm, nil).Once()
	s.node.EXPECT().GetCurrentTerm().Return(s.currentTerm).Once()
	s.node.EXPECT().GetCommitIndex().Return(s.commitIndex).Once()
	s.peerMocks["node2"].EXPECT().AppendEntries(mock.Anything, &raft_service.AppendEntriesRequest{
		Term:         s.currentTerm,
		LeaderId:     s.network.GetId(),
		PrevLogIndex: s.lastIndex,
		PrevLogTerm:  s.lastTerm,
		LeaderCommit: s.commitIndex,
		Entry:        nil,
	}).Return(&raft_service.AppendEntriesResponse{
		Term:    s.lastTerm,
		Success: true,
	}, nil)
	s.node.EXPECT().GetCurrentTerm().Return(s.currentTerm).Once()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.network.electionTimeout)*time.Millisecond)
	defer cancel()
	s.network.SendHeartbeat(ctx, "node2")

	<-ctx.Done()
}

func (s *RaftNetworkTestSuite) TestSendAppendEntriesSuccess() {
	t := s.T()

	s.log.EXPECT().GetLastIndex().Return(s.lastIndex).Once()
	s.log.EXPECT().GetTermAtIndex(s.lastIndex).Return(s.lastTerm, nil).Once()
	s.node.EXPECT().GetCurrentTerm().Return(s.currentTerm).Once()
	s.node.EXPECT().GetCommitIndex().Return(s.commitIndex).Once()
	s.peerMocks["node2"].EXPECT().AppendEntries(mock.Anything, &raft_service.AppendEntriesRequest{
		Term:         s.currentTerm,
		LeaderId:     s.network.GetId(),
		PrevLogIndex: s.lastIndex,
		PrevLogTerm:  s.lastTerm,
		LeaderCommit: s.commitIndex,
		Entry:        &entries.LogEntry{},
	}).Return(&raft_service.AppendEntriesResponse{
		Term:    s.lastTerm,
		Success: true,
	}, nil)
	s.node.EXPECT().GetCurrentTerm().Return(s.currentTerm).Once()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.network.electionTimeout)*time.Millisecond)
	defer cancel()
	status := s.network.SendAppendEntry(ctx, "node2", &entries.LogEntry{})

	<-ctx.Done()

	assert.Equal(t, network.Success, status)
}

func (s *RaftNetworkTestSuite) TestSendAppendEntriesLogInconsistency() {
	t := s.T()

	s.log.EXPECT().GetLastIndex().Return(s.lastIndex).Once()
	s.log.EXPECT().GetTermAtIndex(s.lastIndex).Return(s.lastTerm, nil).Once()
	s.node.EXPECT().GetCurrentTerm().Return(s.currentTerm).Once()
	s.node.EXPECT().GetCommitIndex().Return(s.commitIndex).Once()
	s.peerMocks["node2"].EXPECT().AppendEntries(mock.Anything, &raft_service.AppendEntriesRequest{
		Term:         s.currentTerm,
		LeaderId:     s.network.GetId(),
		PrevLogIndex: s.lastIndex,
		PrevLogTerm:  s.lastTerm,
		LeaderCommit: s.commitIndex,
		Entry:        &entries.LogEntry{},
	}).Return(&raft_service.AppendEntriesResponse{
		Term:    s.lastTerm,
		Success: false,
	}, nil)
	s.node.EXPECT().GetCurrentTerm().Return(s.currentTerm).Once()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.network.electionTimeout)*time.Millisecond)
	defer cancel()
	status := s.network.SendAppendEntry(ctx, "node2", &entries.LogEntry{})

	<-ctx.Done()

	assert.Equal(t, network.LogInconsistency, status)
}

func (s *RaftNetworkTestSuite) TestSendAppendEntriesTimedOut() {
	t := s.T()

	s.log.EXPECT().GetLastIndex().Return(s.lastIndex).Once()
	s.log.EXPECT().GetTermAtIndex(s.lastIndex).Return(s.lastTerm, nil).Once()
	s.node.EXPECT().GetCurrentTerm().Return(s.currentTerm).Once()
	s.node.EXPECT().GetCommitIndex().Return(s.commitIndex).Once()
	waitTimer := time.NewTimer(time.Duration(2*s.network.electionTimeout) * time.Millisecond)
	s.peerMocks["node2"].EXPECT().AppendEntries(mock.Anything, &raft_service.AppendEntriesRequest{
		Term:         s.currentTerm,
		LeaderId:     s.network.GetId(),
		PrevLogIndex: s.lastIndex,
		PrevLogTerm:  s.lastTerm,
		LeaderCommit: s.commitIndex,
		Entry:        &entries.LogEntry{},
	}).Return(&raft_service.AppendEntriesResponse{
		Term:    s.lastTerm,
		Success: false,
	}, nil).WaitUntil(waitTimer.C)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.network.electionTimeout)*time.Millisecond)
	defer cancel()
	status := s.network.SendAppendEntry(ctx, "node2", &entries.LogEntry{})

	<-ctx.Done()

	assert.Equal(t, network.NotReceived, status)
}

func TestRaftNetworkTestSuite(t *testing.T) {
	suite.Run(t, new(RaftNetworkTestSuite))
}
