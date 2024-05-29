package raft_node

import (
	"testing"

	log_mocks "github.com/r-moraru/modular-raft/log/mocks"
	network_mocks "github.com/r-moraru/modular-raft/network/mocks"
	"github.com/r-moraru/modular-raft/node"
	"github.com/r-moraru/modular-raft/proto/entries"
	state_machine_mocks "github.com/r-moraru/modular-raft/state_machine/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestConstructor(t *testing.T) {
	lastIndex := uint64(15)
	lastTerm := uint64(3)
	lastAppliedIndex := uint64(12)
	log := log_mocks.NewLog(t)
	stateMachine := state_machine_mocks.NewStateMachine(t)
	network := network_mocks.NewNetwork(t)

	log.EXPECT().GetLastIndex().Return(lastIndex).Once()
	log.EXPECT().GetTermAtIndex(lastIndex).Return(lastTerm, nil).Once()
	stateMachine.EXPECT().GetLastApplied().Return(lastAppliedIndex)
	network.EXPECT().GetId().Return("node1").Once()

	raftNode, err := New(100, 50, log, stateMachine, network)

	assert.NoError(t, err, "Valid constructor call should not return error.")
	assert.Equal(t, node.Follower, raftNode.GetState())
	assert.Equal(t, lastTerm, raftNode.GetCurrentTerm())
	assert.Equal(t, false, raftNode.VotedForTerm())
	assert.Equal(t, lastAppliedIndex, raftNode.GetCommitIndex())
	assert.Equal(t, lastAppliedIndex, raftNode.lastApplied)
}

func TestBecomesCandidateOnFirstIteration(t *testing.T) {
	lastIndex := uint64(15)
	lastTerm := uint64(3)
	lastAppliedIndex := uint64(12)
	nodeId := "node1"
	log := log_mocks.NewLog(t)
	stateMachine := state_machine_mocks.NewStateMachine(t)
	network := network_mocks.NewNetwork(t)

	log.EXPECT().GetLastIndex().Return(lastIndex).Once()
	log.EXPECT().GetTermAtIndex(lastIndex).Return(lastTerm, nil).Once()
	stateMachine.EXPECT().GetLastApplied().Return(lastAppliedIndex)
	network.EXPECT().GetId().Return(nodeId).Once()

	raftNode, _ := New(100, 50, log, stateMachine, network)
	raftNode.runIteration()

	assert.Equal(t, node.Candidate, raftNode.GetState())
}

func TestSuccessfulCandidateIteration(t *testing.T) {
	lastIndex := uint64(15)
	lastTerm := uint64(3)
	lastAppliedIndex := uint64(12)
	nodeId := "node1"
	peers := []string{
		"node1",
		"node2",
		"node3",
		"node4",
	}
	log := log_mocks.NewLog(t)
	stateMachine := state_machine_mocks.NewStateMachine(t)
	network := network_mocks.NewNetwork(t)

	log.EXPECT().GetLastIndex().Return(lastIndex)
	log.EXPECT().GetTermAtIndex(lastIndex).Return(lastTerm, nil).Once()
	stateMachine.EXPECT().GetLastApplied().Return(lastAppliedIndex)
	network.EXPECT().GetId().Return(nodeId)
	network.EXPECT().SendRequestVoteAsync(lastTerm + 1).Return().Once()
	majorityVoteChan := make(chan bool, 1)
	majorityVoteChan <- true
	network.EXPECT().GotMajorityVote(mock.Anything).Return(majorityVoteChan).Once()
	network.EXPECT().GetPeerList().Return(peers).Once()

	raftNode, _ := New(100, 50, log, stateMachine, network)
	raftNode.runIteration()
	raftNode.runIteration()

	assert.Equal(t, node.Leader, raftNode.GetState())
	assert.Equal(t, lastTerm+1, raftNode.GetCurrentTerm())
	assert.Equal(t, false, raftNode.VotedForTerm())
}

func TestFailedCandidateIterationTurnsNodeToFollower(t *testing.T) {
	lastIndex := uint64(15)
	lastTerm := uint64(3)
	lastAppliedIndex := uint64(12)
	nodeId := "node1"
	log := log_mocks.NewLog(t)
	stateMachine := state_machine_mocks.NewStateMachine(t)
	network := network_mocks.NewNetwork(t)

	log.EXPECT().GetLastIndex().Return(lastIndex)
	log.EXPECT().GetTermAtIndex(lastIndex).Return(lastTerm, nil).Once()
	stateMachine.EXPECT().GetLastApplied().Return(lastAppliedIndex)
	network.EXPECT().GetId().Return(nodeId)
	network.EXPECT().SendRequestVoteAsync(lastTerm + 1).Return().Once()
	majorityVoteChan := make(chan bool, 2)
	majorityVoteChan <- false
	network.EXPECT().GotMajorityVote(mock.Anything).Return(majorityVoteChan).Once()

	raftNode, _ := New(100, 50, log, stateMachine, network)
	raftNode.runIteration()
	raftNode.runIteration()

	assert.Equal(t, node.Follower, raftNode.GetState())
	assert.Equal(t, lastTerm+1, raftNode.GetCurrentTerm())
	assert.Equal(t, false, raftNode.VotedForTerm())
}

func TestTimedOutCandidateIterationRetries(t *testing.T) {
	lastIndex := uint64(15)
	lastTerm := uint64(3)
	lastAppliedIndex := uint64(12)
	nodeId := "node1"
	peers := []string{
		"node1",
		"node2",
		"node3",
		"node4",
	}
	log := log_mocks.NewLog(t)
	stateMachine := state_machine_mocks.NewStateMachine(t)
	network := network_mocks.NewNetwork(t)

	log.EXPECT().GetLastIndex().Return(lastIndex)
	log.EXPECT().GetTermAtIndex(lastIndex).Return(lastTerm, nil).Once()
	stateMachine.EXPECT().GetLastApplied().Return(lastAppliedIndex)
	network.EXPECT().GetId().Return(nodeId)
	network.EXPECT().SendRequestVoteAsync(lastTerm + 1).Return().Once()
	network.EXPECT().SendRequestVoteAsync(lastTerm + 2).Return().Once()
	majorityVoteChan := make(chan bool, 1)
	network.EXPECT().GotMajorityVote(mock.Anything).Return(majorityVoteChan).Twice()
	network.EXPECT().GetPeerList().Return(peers).Once()

	raftNode, _ := New(10, 5, log, stateMachine, network)
	raftNode.runIteration()
	// one timeout
	raftNode.runIteration()
	// one successful vote
	majorityVoteChan <- true
	raftNode.runIteration()

	assert.Equal(t, node.Leader, raftNode.GetState())
	assert.Equal(t, lastTerm+2, raftNode.GetCurrentTerm())
	assert.Equal(t, false, raftNode.VotedForTerm())

}

func TestAppliesCommittedEntriesNotYetApplied(t *testing.T) {
	lastIndex := uint64(15)
	lastTerm := uint64(3)
	lastAppliedIndex := uint64(12)
	nodeId := "node1"
	log := log_mocks.NewLog(t)
	stateMachine := state_machine_mocks.NewStateMachine(t)
	network := network_mocks.NewNetwork(t)
	entry := &entries.LogEntry{
		Index:           13,
		Term:            3,
		ClientID:        "client-id",
		SerializationID: 12345,
	}

	log.EXPECT().GetLastIndex().Return(lastIndex).Once()
	log.EXPECT().GetTermAtIndex(lastIndex).Return(lastTerm, nil).Once()
	log.EXPECT().GetEntry(uint64(13)).Return(entry, nil).Once()
	stateMachine.EXPECT().GetLastApplied().Return(lastAppliedIndex)
	stateMachine.EXPECT().Apply(mock.Anything).Return(nil).Once()
	network.EXPECT().GetId().Return(nodeId).Once()

	raftNode, _ := New(10, 5, log, stateMachine, network)
	raftNode.SetCommitIndex(raftNode.GetCommitIndex() + 1)
	raftNode.runIteration()

	assert.Equal(t, node.Candidate, raftNode.GetState())
	assert.Equal(t, uint64(13), raftNode.lastApplied)
}

func TestMultipleFollowerIterations(t *testing.T) {
	lastIndex := uint64(16)
	lastTerm := uint64(4)
	lastAppliedIndex := uint64(12)
	nodeId := "node1"
	log := log_mocks.NewLog(t)
	stateMachine := state_machine_mocks.NewStateMachine(t)
	network := network_mocks.NewNetwork(t)
	entries := []*entries.LogEntry{
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

	log.EXPECT().GetLastIndex().Return(lastIndex).Once()
	log.EXPECT().GetTermAtIndex(lastIndex).Return(lastTerm, nil).Once()
	log.EXPECT().GetEntry(uint64(13)).Return(entries[0], nil).Once()
	log.EXPECT().GetEntry(uint64(14)).Return(entries[1], nil).Once()
	log.EXPECT().GetEntry(uint64(15)).Return(entries[2], nil).Once()
	log.EXPECT().GetEntry(uint64(16)).Return(entries[3], nil).Once()
	stateMachine.EXPECT().GetLastApplied().Return(lastAppliedIndex)
	stateMachine.EXPECT().Apply(entries[0]).Return(nil).Once()
	stateMachine.EXPECT().Apply(entries[1]).Return(nil).Once()
	stateMachine.EXPECT().Apply(entries[2]).Return(nil).Once()
	stateMachine.EXPECT().Apply(entries[3]).Return(nil).Once()
	network.EXPECT().SendRequestVoteAsync(lastTerm + 1).Return().Once()
	majorityVoteChan := make(chan bool, 1)
	majorityVoteChan <- false
	network.EXPECT().GotMajorityVote(mock.Anything).Return(majorityVoteChan).Once()
	network.EXPECT().GetId().Return(nodeId)

	raftNode, _ := New(500, 50, log, stateMachine, network)
	raftNode.SetCommitIndex(raftNode.GetCommitIndex() + 4)
	raftNode.runIteration()
	raftNode.runIteration()
	raftNode.runIteration()
	raftNode.runIteration()

	assert.Equal(t, node.Follower, raftNode.GetState())
	assert.Equal(t, uint64(16), raftNode.lastApplied)
}

// TODO: test leader iteration + cases
