package raft_node

import (
	"errors"
	"testing"
	"time"

	log_mocks "github.com/r-moraru/modular-raft/log/mocks"
	"github.com/r-moraru/modular-raft/network"
	network_mocks "github.com/r-moraru/modular-raft/network/mocks"
	"github.com/r-moraru/modular-raft/node"
	"github.com/r-moraru/modular-raft/proto/entries"
	state_machine_mocks "github.com/r-moraru/modular-raft/state_machine/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type RaftNodeTestSuite struct {
	suite.Suite

	lastIndex        uint64
	lastTerm         uint64
	lastAppliedIndex uint64
	nodeId           string
	peers            []string
	entries          []*entries.LogEntry
	log              *log_mocks.Log
	network          *network_mocks.Network
	stateMachine     *state_machine_mocks.StateMachine
}

func (s *RaftNodeTestSuite) SetupTest() {
	s.log = log_mocks.NewLog(s.T())
	s.stateMachine = state_machine_mocks.NewStateMachine(s.T())
	s.network = network_mocks.NewNetwork(s.T())
	s.entries = []*entries.LogEntry{
		{
			Index:           13,
			Term:            3,
			ClientID:        "client-id",
			SerializationID: 12345,
		},
		{
			Index:           14,
			Term:            3,
			ClientID:        "client-id",
			SerializationID: 12346,
		},
		{
			Index:           15,
			Term:            3,
			ClientID:        "client-id",
			SerializationID: 12347,
		},
		{
			Index:           16,
			Term:            3,
			ClientID:        "client-id",
			SerializationID: 12348,
		},
	}

}

func (s *RaftNodeTestSuite) mockNodeConstructorCalls() {
	s.log.EXPECT().GetLength().Return(uint64(16)).Once()
	s.log.EXPECT().GetLastIndex().Return(s.lastIndex).Once()
	s.log.EXPECT().GetTermAtIndex(s.lastIndex).Return(s.lastTerm, nil).Once()
	s.stateMachine.EXPECT().GetLastApplied().Return(s.lastAppliedIndex).Twice()
	s.network.EXPECT().GetId().Return(s.nodeId).Once()
}

func (s *RaftNodeTestSuite) mockSuccessfulCandidateIterationCalls(term uint64) {
	majorityVoteChan := make(chan bool, 1)
	majorityVoteChan <- true
	s.network.EXPECT().GetId().Return(s.nodeId)
	s.network.EXPECT().SendRequestVote(mock.Anything, term).Return(majorityVoteChan).Once()
}

func (s *RaftNodeTestSuite) mockFailedCandidateIterationCalls(term uint64) {
	majorityVoteChan := make(chan bool, 1)
	majorityVoteChan <- false
	s.network.EXPECT().GetId().Return(s.nodeId)
	s.network.EXPECT().SendRequestVote(mock.Anything, term).Return(majorityVoteChan).Once()
}

func (s *RaftNodeTestSuite) mockTimedOutCandidateIterationCalls(term uint64) {
	majorityVoteChan := make(chan bool, 1)
	s.network.EXPECT().GetId().Return(s.nodeId)
	s.network.EXPECT().SendRequestVote(mock.Anything, term).Return(majorityVoteChan).Once()
}

func (s *RaftNodeTestSuite) mockResetLeaderBookkeepingCalls() {
	s.log.EXPECT().GetLastIndex().Return(s.lastIndex).Once()
	s.network.EXPECT().GetPeerList().Return(s.peers).Once()
}

func (s *RaftNodeTestSuite) mockEntryApplicationCalls(i int) {
	s.stateMachine.EXPECT().GetLastApplied().Return(s.lastAppliedIndex)
	s.log.EXPECT().GetEntry(s.entries[i].Index).Return(s.entries[i], nil).Once()
	s.stateMachine.EXPECT().Apply(s.entries[i]).Return(nil).Once()
}

func (s *RaftNodeTestSuite) mockSuccessfulLeaderReplicationCalls(peerIndex int, entryIndex int) {
	s.log.EXPECT().GetLastIndex().Return(s.lastIndex)
	s.log.EXPECT().GetEntry(s.entries[entryIndex].Index).Return(s.entries[entryIndex], nil).Once()
	s.network.EXPECT().SendAppendEntry(mock.Anything, s.peers[peerIndex], s.entries[entryIndex]).Return(network.Success).Once()
}

func (s *RaftNodeTestSuite) mockCommitIndexUpdateOutOfBoundsCalls() {
	s.log.EXPECT().GetTermAtIndex(s.lastIndex+1).Return(0, errors.New("Index out of bounds"))
}

func (s *RaftNodeTestSuite) mockLeaderHeartbeatCalls(peerIndex int) {
	// TODO: mock this properly
	s.network.EXPECT().SendHeartbeat(mock.Anything, s.peers[peerIndex], 0).Once()
}

func (s *RaftNodeTestSuite) TestConstructor() {
	t := s.T()
	s.lastIndex = uint64(15)
	s.lastTerm = uint64(3)
	s.lastAppliedIndex = uint64(12)

	s.mockNodeConstructorCalls()

	raftNode, err := New(10, 5, s.log, s.stateMachine, s.network)
	<-time.After(5 * time.Millisecond)

	assert.NoError(t, err, "Valid constructor call should not return error.")
	assert.Equal(t, node.Follower, raftNode.GetState())
	assert.Equal(t, s.lastTerm, raftNode.GetCurrentTerm())
	assert.Equal(t, false, raftNode.VotedForTerm())
	assert.Equal(t, s.lastAppliedIndex, raftNode.GetCommitIndex())
	assert.Equal(t, s.lastAppliedIndex, raftNode.lastApplied)
}

func (s *RaftNodeTestSuite) TestBecomesCandidateOnFirstIteration() {
	t := s.T()

	s.lastIndex = uint64(15)
	s.lastTerm = uint64(3)
	s.lastAppliedIndex = uint64(12)
	s.nodeId = "node1"

	s.mockNodeConstructorCalls()

	raftNode, _ := New(10, 5, s.log, s.stateMachine, s.network)
	<-time.After(5 * time.Millisecond)
	raftNode.runIteration()

	assert.Equal(t, node.Candidate, raftNode.GetState())
}

func (s *RaftNodeTestSuite) TestSuccessfulCandidateIteration() {
	t := s.T()

	s.lastIndex = uint64(15)
	s.lastTerm = uint64(3)
	s.lastAppliedIndex = uint64(12)
	s.nodeId = "node1"
	s.peers = []string{
		"node2",
		"node3",
		"node4",
		"node5",
	}

	s.mockNodeConstructorCalls()
	s.mockSuccessfulCandidateIterationCalls(s.lastTerm + 1)
	s.mockResetLeaderBookkeepingCalls()

	raftNode, _ := New(10, 5, s.log, s.stateMachine, s.network)
	<-time.After(5 * time.Millisecond)
	raftNode.runIteration()
	raftNode.runIteration()

	assert.Equal(t, node.Leader, raftNode.GetState())
	assert.Equal(t, s.lastTerm+1, raftNode.GetCurrentTerm())
	assert.Equal(t, false, raftNode.VotedForTerm())
}

func (s *RaftNodeTestSuite) TestFailedCandidateIterationTurnsNodeToFollower() {
	t := s.T()

	s.lastIndex = uint64(15)
	s.lastTerm = uint64(3)
	s.lastAppliedIndex = uint64(12)
	s.nodeId = "node1"

	s.mockNodeConstructorCalls()
	s.mockFailedCandidateIterationCalls(s.lastTerm + 1)

	raftNode, _ := New(10, 5, s.log, s.stateMachine, s.network)
	<-time.After(5 * time.Millisecond)
	raftNode.runIteration()
	raftNode.runIteration()

	assert.Equal(t, node.Follower, raftNode.GetState())
	assert.Equal(t, s.lastTerm+1, raftNode.GetCurrentTerm())
	assert.Equal(t, false, raftNode.VotedForTerm())
}

func (s *RaftNodeTestSuite) TestTimedOutCandidateIterationRetries() {
	t := s.T()

	s.lastIndex = uint64(15)
	s.lastTerm = uint64(3)
	s.lastAppliedIndex = uint64(12)
	s.nodeId = "node1"
	s.peers = []string{
		"node2",
		"node3",
		"node4",
		"node5",
	}

	s.mockNodeConstructorCalls()
	s.mockTimedOutCandidateIterationCalls(s.lastTerm + 1)
	s.mockSuccessfulCandidateIterationCalls(s.lastTerm + 2)
	s.mockResetLeaderBookkeepingCalls()

	raftNode, _ := New(10, 5, s.log, s.stateMachine, s.network)
	<-time.After(5 * time.Millisecond)
	raftNode.runIteration()
	raftNode.runIteration()
	raftNode.runIteration()

	assert.Equal(t, node.Leader, raftNode.GetState())
	assert.Equal(t, s.lastTerm+2, raftNode.GetCurrentTerm())
	assert.Equal(t, false, raftNode.VotedForTerm())
}

func (s *RaftNodeTestSuite) TestAppliesCommittedEntriesNotYetApplied() {
	t := s.T()

	s.lastIndex = uint64(15)
	s.lastTerm = uint64(3)
	s.lastAppliedIndex = uint64(12)
	s.nodeId = "node1"
	s.entries = []*entries.LogEntry{
		{
			Index:           13,
			Term:            3,
			ClientID:        "client-id",
			SerializationID: 12345,
		},
	}

	s.mockNodeConstructorCalls()
	s.mockEntryApplicationCalls(0)

	raftNode, _ := New(10, 5, s.log, s.stateMachine, s.network)
	<-time.After(5 * time.Millisecond)
	raftNode.SetCommitIndex(raftNode.GetCommitIndex() + 1)
	raftNode.runIteration()

	assert.Equal(t, node.Candidate, raftNode.GetState())
	assert.Equal(t, s.entries[0].Index, raftNode.lastApplied)
}

func (s *RaftNodeTestSuite) TestMultipleFollowerIterations() {
	t := s.T()

	s.lastIndex = uint64(16)
	s.lastTerm = uint64(4)
	s.lastAppliedIndex = uint64(12)
	s.nodeId = "node1"

	s.mockNodeConstructorCalls()
	s.mockFailedCandidateIterationCalls(s.lastTerm + 1)
	s.mockEntryApplicationCalls(0)
	s.mockEntryApplicationCalls(1)
	s.mockEntryApplicationCalls(2)
	s.mockEntryApplicationCalls(3)

	raftNode, _ := New(10, 5, s.log, s.stateMachine, s.network)
	<-time.After(5 * time.Millisecond)
	raftNode.SetCommitIndex(raftNode.GetCommitIndex() + 4)
	raftNode.runIteration()
	raftNode.runIteration()
	raftNode.runIteration()
	raftNode.runIteration()

	assert.Equal(t, node.Follower, raftNode.GetState())
	assert.Equal(t, uint64(16), raftNode.lastApplied)
}

func (s *RaftNodeTestSuite) TestLeaderIterationSendsAppendEntriesOnFirstIteration() {
	t := s.T()

	s.lastIndex = uint64(13)
	s.lastTerm = uint64(3)
	s.lastAppliedIndex = uint64(13)
	s.nodeId = "node1"
	s.peers = []string{
		"node1",
		"node2",
	}

	s.mockNodeConstructorCalls()
	s.mockSuccessfulCandidateIterationCalls(s.lastTerm + 1)
	s.mockResetLeaderBookkeepingCalls()

	s.mockSuccessfulLeaderReplicationCalls(0, 1)
	s.mockSuccessfulLeaderReplicationCalls(1, 1)
	s.mockCommitIndexUpdateOutOfBoundsCalls()

	raftNode, _ := New(10, 5, s.log, s.stateMachine, s.network)
	<-time.After(5 * time.Millisecond)
	raftNode.runIteration()
	raftNode.runIteration()
	raftNode.runIteration()

	assert.Equal(t, node.Leader, raftNode.GetState())
	val, ok := raftNode.matchIndex.Load(s.peers[0])
	assert.True(t, ok, "Leader bookkeeping incomplete.")
	assert.Equal(t, val.(uint64), s.lastIndex+1)
	val, ok = raftNode.matchIndex.Load(s.peers[1])
	assert.True(t, ok, "Leader bookkeeping incomplete.")
	assert.Equal(t, val.(uint64), s.lastIndex+1)
	val, ok = raftNode.nextIndex.Load(s.peers[0])
	assert.True(t, ok, "Leader bookkeeping incomplete.")
	assert.Equal(t, val.(uint64), s.lastIndex+2)
	val, ok = raftNode.nextIndex.Load(s.peers[1])
	assert.True(t, ok, "Leader bookkeeping incomplete.")
	assert.Equal(t, val.(uint64), s.lastIndex+2)
}

func (s *RaftNodeTestSuite) TestLeaderIterationSendsHeartbeats() {
	t := s.T()

	s.lastIndex = uint64(13)
	s.lastTerm = uint64(3)
	s.lastAppliedIndex = uint64(13)
	s.nodeId = "node1"
	s.peers = []string{
		"node1",
		"node2",
	}

	s.mockNodeConstructorCalls()
	s.mockSuccessfulCandidateIterationCalls(s.lastTerm + 1)
	s.mockResetLeaderBookkeepingCalls()
	s.mockSuccessfulLeaderReplicationCalls(0, 0)
	s.mockSuccessfulLeaderReplicationCalls(1, 0)
	s.mockCommitIndexUpdateOutOfBoundsCalls()

	// second and third iteration
	s.mockLeaderHeartbeatCalls(0)
	s.mockLeaderHeartbeatCalls(1)
	s.mockLeaderHeartbeatCalls(0)
	s.mockLeaderHeartbeatCalls(1)

	raftNode, _ := New(10, 5, s.log, s.stateMachine, s.network)
	<-time.After(5 * time.Millisecond)
	raftNode.runIteration()
	raftNode.runIteration()
	raftNode.runIteration()
	raftNode.runIteration()
	raftNode.runIteration()

	assert.Equal(t, node.Leader, raftNode.GetState())
	val, ok := raftNode.matchIndex.Load(s.peers[0])
	assert.True(t, ok, "Leader bookkeeping incomplete.")
	assert.Equal(t, val.(uint64), s.lastIndex+1)
	val, ok = raftNode.matchIndex.Load(s.peers[1])
	assert.True(t, ok, "Leader bookkeeping incomplete.")
	assert.Equal(t, val.(uint64), s.lastIndex+1)
	val, ok = raftNode.nextIndex.Load(s.peers[0])
	assert.True(t, ok, "Leader bookkeeping incomplete.")
	assert.Equal(t, val.(uint64), s.lastIndex+1)
	val, ok = raftNode.nextIndex.Load(s.peers[1])
	assert.True(t, ok, "Leader bookkeeping incomplete.")
	assert.Equal(t, val.(uint64), s.lastIndex+1)
}

// // TODO: test leader steps down, adjusts bookkeeping on peer failures,
// // finds next commit index

func TestRaftNodeTestSuite(t *testing.T) {
	suite.Run(t, new(RaftNodeTestSuite))
}
