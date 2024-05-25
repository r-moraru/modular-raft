package node

import (
	"context"
	logger "log"
	"time"

	"github.com/r-moraru/modular-raft/log"
	"github.com/r-moraru/modular-raft/network"
	"github.com/r-moraru/modular-raft/state_machine"
)

type State int64

const (
	Leader State = iota
	Follower
	Candidate
)

type Node struct {
	state           State
	timer           time.Timer
	electionTimeout int64
	heartbeat       int64

	currentTerm int64
	votedFor    network.PeerId
	commitIndex int64
	lastApplied int64
	nextIndex   map[network.PeerId]int64
	matchIndex  map[network.PeerId]int64

	log          log.Log
	stateMachine state_machine.StateMachine
	network      network.Network
}

func (n *Node) resetTimer() {
	n.timer.Reset(time.Duration(n.electionTimeout) * time.Millisecond)
}

func (n *Node) runFollowerIteration() {
	entry := n.network.GetNewLogEntry()
	if entry != nil {
		n.log.InsertLogEntry(entry)
	}

	n.commitIndex = n.network.UpdateCommitIndex(n.commitIndex)

	select {
	case <-n.timer.C:
		n.state = Candidate
	default:
		return
	}
}

func (n *Node) runCandidateIteration() {
	n.currentTerm += 1
	n.votedFor = n.network.GetId()
	n.resetTimer()
	n.network.SendRequestVoteAsync(n.currentTerm)
	select {
	// TODO(Optional): also check network for received AppendEntriesRPC from new leader
	case <-n.network.GotMajorityVote():
		n.state = Leader
	case <-n.timer.C:
		return
	}
}

func (n *Node) runLeaderIteration() {
	request := n.network.GetRequest()
	if request != nil {
		n.log.AppendEntry(n.currentTerm, request)
	}

	numReadyToCommitNext := 0
	for peerId, index := range n.nextIndex {
		peerResponse := n.network.GetPeerResponse(index, n.log.GetTermAtIndex(index), peerId)
		switch peerResponse {
		case network.NotReceived:
			continue
		case network.Success:
			n.matchIndex[peerId] = n.nextIndex[peerId]
			n.nextIndex[peerId] += 1
		case network.LogInconsistency:
			n.nextIndex[peerId] -= 1
		}

		if request == nil {
			n.network.SendHeartbeatAsync(peerId)
		} else {
			n.network.SendAppendEntryAsync(peerId, n.log.GetEntry(index))
		}

		if n.matchIndex[peerId] > n.commitIndex && n.log.GetEntry(n.commitIndex+1).Term == n.currentTerm {
			numReadyToCommitNext += 1
		}
		if numReadyToCommitNext > len(n.matchIndex) {
			n.commitIndex += 1
		}
	}

	time.Sleep(time.Duration(n.heartbeat) * time.Millisecond)
}

func (n *Node) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if n.network.CheckUpdateTimer() {
				n.resetTimer()
			}
			networkGreatestTerm := n.network.CheckGreatestTerm()
			if networkGreatestTerm > n.currentTerm {
				n.state = Follower
			}

			switch n.state {
			case Follower:
				n.runFollowerIteration()
			case Candidate:
				n.runCandidateIteration()
			case Leader:
				n.runLeaderIteration()
			}

			if n.commitIndex > n.lastApplied {
				logEntry := n.log.GetEntry(n.lastApplied + 1)
				err := n.stateMachine.Apply(logEntry.Entry)
				if err != nil {
					logger.Fatalf("State machine unable to apply entry at index %d, term %d.", logEntry.Index, logEntry.Term)
				}
				n.lastApplied += 1
			}
		}
	}
}
