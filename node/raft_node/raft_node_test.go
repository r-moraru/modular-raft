package raft_node

import (
	"testing"

	log_mocks "github.com/r-moraru/modular-raft/log/mocks"
	network_mocks "github.com/r-moraru/modular-raft/network/mocks"
	"github.com/r-moraru/modular-raft/node"
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
	assert.Equal(t, false, raftNode.VotedForTerm(lastTerm))
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
	assert.Equal(t, false, raftNode.VotedForTerm(lastTerm+1))
}
