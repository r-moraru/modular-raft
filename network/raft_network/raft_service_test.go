package raft_network

import (
	"context"
	"errors"
	"testing"

	log_mocks "github.com/r-moraru/modular-raft/log/mocks"
	"github.com/r-moraru/modular-raft/node"
	node_mocks "github.com/r-moraru/modular-raft/node/mocks"
	"github.com/r-moraru/modular-raft/proto/entries"
	"github.com/r-moraru/modular-raft/proto/raft_service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RaftServiceTestSuite struct {
	suite.Suite

	raftNode *node_mocks.Node
	log      *log_mocks.Log
	service  *RaftService
}

func (s *RaftServiceTestSuite) SetupTest() {
	s.log = log_mocks.NewLog(s.T())
	s.raftNode = node_mocks.NewNode(s.T())
	s.service = &RaftService{
		RaftNode: s.raftNode,
		Log:      s.log,
	}
}

func (s *RaftServiceTestSuite) TestAppendEntriesRepliesFalseIfLeaderTermIsBehind() {
	t := s.T()

	currentTerm := uint64(4)
	s.raftNode.EXPECT().GetCurrentTerm().Return(currentTerm)
	req := &raft_service.AppendEntriesRequest{
		Term:  3,
		Entry: &entries.LogEntry{},
	}

	ctx := context.Background()
	res, err := s.service.AppendEntries(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, false, res.Success)
	assert.Equal(t, currentTerm, res.Term)
}

func (s *RaftServiceTestSuite) TestAppendEntriesNoEntryAtPrevLogIndex() {
	t := s.T()

	currentTerm := uint64(2)
	leaderId := "leader"
	s.raftNode.EXPECT().GetCurrentTerm().Return(currentTerm)
	s.raftNode.EXPECT().SetCurrentLeaderID(leaderId)
	s.raftNode.EXPECT().GetState().Return(node.Follower).Once()
	s.raftNode.EXPECT().ResetTimer().Once()
	s.log.EXPECT().GetLength().Return(14)
	s.raftNode.EXPECT().GetCommitIndex().Return(14).Once()
	req := &raft_service.AppendEntriesRequest{
		Term:         currentTerm,
		LeaderId:     leaderId,
		PrevLogIndex: 15,
		Entry:        &entries.LogEntry{},
	}

	ctx := context.Background()
	res, err := s.service.AppendEntries(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, false, res.Success)
	assert.Equal(t, currentTerm, res.Term)
}

func (s *RaftServiceTestSuite) TestAppendEntriesTurnsNodeToFollowerIfTermPasses() {
	t := s.T()

	currentTerm := uint64(2)
	leaderId := "leader"
	s.raftNode.EXPECT().GetCurrentTerm().Return(currentTerm)
	s.raftNode.EXPECT().SetCurrentLeaderID(leaderId)
	s.raftNode.EXPECT().GetState().Return(node.Leader).Once()
	s.raftNode.EXPECT().SetState(node.Follower).Once()
	s.raftNode.EXPECT().ClearVotedFor().Once()
	s.raftNode.EXPECT().SetCurrentTerm(currentTerm)
	s.raftNode.EXPECT().ResetTimer().Once()
	s.raftNode.EXPECT().GetCommitIndex().Return(14).Once()
	s.log.EXPECT().GetLength().Return(14)
	req := &raft_service.AppendEntriesRequest{
		Term:         currentTerm,
		LeaderId:     leaderId,
		PrevLogIndex: 15,
		Entry:        &entries.LogEntry{},
	}

	ctx := context.Background()
	res, err := s.service.AppendEntries(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, false, res.Success)
	assert.Equal(t, currentTerm, res.Term)
}

func (s *RaftServiceTestSuite) TestAppendEntriesThrowsErrorIfLogCannotGetTerm() {
	t := s.T()

	prevLogIndex := uint64(15)
	currentTerm := uint64(2)
	leaderId := "leader"
	s.raftNode.EXPECT().GetCurrentTerm().Return(currentTerm)
	s.raftNode.EXPECT().SetCurrentLeaderID(leaderId)
	s.raftNode.EXPECT().GetState().Return(node.Follower).Once()
	s.raftNode.EXPECT().ResetTimer().Once()
	s.log.EXPECT().GetLength().Return(prevLogIndex)
	s.raftNode.EXPECT().GetCommitIndex().Return(14).Once()
	s.log.EXPECT().GetTermAtIndex(prevLogIndex).Return(0, errors.New("error"))
	req := &raft_service.AppendEntriesRequest{
		Term:         currentTerm,
		LeaderId:     leaderId,
		PrevLogIndex: prevLogIndex,
		Entry:        &entries.LogEntry{},
	}

	ctx := context.Background()
	res, err := s.service.AppendEntries(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, false, res.Success)
	assert.Equal(t, currentTerm, res.Term)
}

func (s *RaftServiceTestSuite) TestAppendEntriesReturnsFalseIfPrevTermMismatch() {
	t := s.T()

	prevLogIndex := uint64(15)
	currentTerm := uint64(2)
	leaderId := "leader"
	s.raftNode.EXPECT().GetCurrentTerm().Return(currentTerm)
	s.raftNode.EXPECT().SetCurrentLeaderID(leaderId)
	s.raftNode.EXPECT().GetState().Return(node.Follower).Once()
	s.raftNode.EXPECT().ResetTimer().Once()
	s.log.EXPECT().GetLength().Return(prevLogIndex)
	s.log.EXPECT().GetTermAtIndex(prevLogIndex).Return(currentTerm-1, nil)
	s.raftNode.EXPECT().GetCommitIndex().Return(14).Once()
	req := &raft_service.AppendEntriesRequest{
		Term:         currentTerm,
		LeaderId:     leaderId,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  currentTerm,
		Entry:        &entries.LogEntry{},
	}

	ctx := context.Background()
	res, err := s.service.AppendEntries(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, false, res.Success)
	assert.Equal(t, currentTerm, res.Term)
}

func (s *RaftServiceTestSuite) TestAppendEntriesAcceptsHeartbeat() {
	t := s.T()

	prevLogIndex := uint64(15)
	currentTerm := uint64(2)
	leaderId := "leader"
	s.raftNode.EXPECT().GetCurrentTerm().Return(currentTerm)
	s.raftNode.EXPECT().SetCurrentLeaderID(leaderId)
	s.raftNode.EXPECT().GetState().Return(node.Follower).Once()
	s.raftNode.EXPECT().ResetTimer().Once()
	s.log.EXPECT().GetLength().Return(prevLogIndex)
	s.log.EXPECT().GetTermAtIndex(prevLogIndex).Return(currentTerm, nil)
	s.raftNode.EXPECT().GetCommitIndex().Return(14).Once()
	req := &raft_service.AppendEntriesRequest{
		Term:         currentTerm,
		LeaderId:     leaderId,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  currentTerm,
	}

	ctx := context.Background()
	res, err := s.service.AppendEntries(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, true, res.Success)
	assert.Equal(t, currentTerm, res.Term)
}

func (s *RaftServiceTestSuite) TestAppendEntriesReturnsSuccessIfItAleadyHasTheEntry() {
	t := s.T()

	prevLogIndex := uint64(15)
	currentTerm := uint64(2)
	leaderId := "leader"
	s.raftNode.EXPECT().GetCurrentTerm().Return(currentTerm)
	s.raftNode.EXPECT().SetCurrentLeaderID(leaderId)
	s.raftNode.EXPECT().GetState().Return(node.Follower).Once()
	s.raftNode.EXPECT().ResetTimer().Once()
	s.log.EXPECT().GetLength().Return(prevLogIndex)
	s.log.EXPECT().GetTermAtIndex(prevLogIndex).Return(currentTerm, nil).Once()
	s.log.EXPECT().GetLastIndex().Return(prevLogIndex + 1)
	s.log.EXPECT().GetTermAtIndex(prevLogIndex+1).Return(currentTerm, nil).Once()
	s.raftNode.EXPECT().GetCommitIndex().Return(14).Once()
	req := &raft_service.AppendEntriesRequest{
		Term:         currentTerm,
		LeaderId:     leaderId,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  currentTerm,
		Entry: &entries.LogEntry{
			Term:  currentTerm,
			Index: prevLogIndex + 1,
		},
	}

	ctx := context.Background()
	res, err := s.service.AppendEntries(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, true, res.Success)
	assert.Equal(t, currentTerm, res.Term)
}

func (s *RaftServiceTestSuite) TestAppendEntriesAppendToLog() {
	t := s.T()

	prevLogIndex := uint64(15)
	lastCommit := uint64(15)
	currentTerm := uint64(2)
	leaderId := "leader"
	entry := &entries.LogEntry{
		Term:  currentTerm,
		Index: prevLogIndex + 1,
	}
	s.raftNode.EXPECT().GetCurrentTerm().Return(currentTerm)
	s.raftNode.EXPECT().SetCurrentLeaderID(leaderId)
	s.raftNode.EXPECT().GetState().Return(node.Follower).Once()
	s.raftNode.EXPECT().ResetTimer().Once()
	s.log.EXPECT().GetLength().Return(prevLogIndex)
	s.log.EXPECT().GetTermAtIndex(prevLogIndex).Return(currentTerm, nil).Once()
	s.log.EXPECT().GetLastIndex().Return(prevLogIndex)
	s.raftNode.EXPECT().GetCommitIndex().Return(lastCommit).Once()
	s.log.EXPECT().InsertLogEntry(entry).Return(nil).Once()
	req := &raft_service.AppendEntriesRequest{
		Term:         currentTerm,
		LeaderId:     leaderId,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  currentTerm,
		LeaderCommit: lastCommit,
		Entry:        entry,
	}

	ctx := context.Background()
	res, err := s.service.AppendEntries(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, true, res.Success)
	assert.Equal(t, currentTerm, res.Term)
}

func TestRaftServiceTestSuite(t *testing.T) {
	suite.Run(t, new(RaftServiceTestSuite))
}
