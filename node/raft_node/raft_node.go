package raft_node

import (
	"context"
	logger "log"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/any"
	"github.com/r-moraru/modular-raft/log"
	"github.com/r-moraru/modular-raft/network"
	"github.com/r-moraru/modular-raft/node"
	"github.com/r-moraru/modular-raft/proto/entries"
	"github.com/r-moraru/modular-raft/state_machine"
)

type Node struct {
	state           node.State
	stateLock       *sync.RWMutex
	timer           time.Timer
	electionTimeout uint64
	heartbeat       uint64

	currentTerm     uint64
	currentTermLock *sync.RWMutex
	votedFor        string
	votedForLock    *sync.RWMutex
	commitIndex     uint64
	lastApplied     uint64
	nextIndex       map[string]uint64
	matchIndex      map[string]uint64

	log          log.Log
	stateMachine state_machine.StateMachine
	network      network.Network
}

func (n *Node) GetState() node.State {
	n.stateLock.RLock()
	defer n.stateLock.RUnlock()
	return n.state
}

func (n *Node) SetState(newState node.State) {
	n.stateLock.Lock()
	defer n.stateLock.Unlock()
	n.state = newState
}

func (n *Node) GetVotedFor() string {
	n.votedForLock.RLock()
	defer n.votedForLock.RUnlock()
	return n.votedFor
}

func (n *Node) SetVotedFor(peerId string) {
	n.votedForLock.Lock()
	defer n.votedForLock.Unlock()
	n.votedFor = peerId
}

func (n *Node) GetCurrentTerm() uint64 {
	n.currentTermLock.RLock()
	defer n.currentTermLock.RUnlock()
	return n.currentTerm
}

func (n *Node) SetCurrentTerm(newTerm uint64) {
	n.currentTermLock.Lock()
	defer n.currentTermLock.Unlock()
	n.currentTerm = newTerm
}

func (n *Node) SetVotedForTerm(term uint64, voted bool) {
	// TODO: implement voted for term
}

func (n *Node) resetTimer() {
	n.timer.Reset(time.Duration(n.electionTimeout) * time.Millisecond)
}

func (n *Node) runFollowerIteration() {
	entry := n.network.GetNewLogEntry()
	if entry != nil {
		n.log.InsertLogEntry(entry)
	}

	n.commitIndex = n.network.GetUpdatedCommitIndex(n.commitIndex)

	select {
	case <-n.timer.C:
		n.SetState(node.Candidate)
	default:
		return
	}
}

func (n *Node) runCandidateIteration() {
	n.SetCurrentTerm(n.GetCurrentTerm() + 1)
	n.SetVotedFor(n.network.GetId())
	n.resetTimer()
	n.network.SendRequestVoteAsync(n.GetCurrentTerm())
	select {
	// TODO(Optional): also check network for received AppendEntriesRPC from new leader
	case <-n.network.GotMajorityVote():
		n.SetState(node.Leader)
		// TODO: commit blank no-op to prevent stale reads
	case <-n.timer.C:
		return
	}
}

func (n *Node) runLeaderIteration() {
	numReadyToCommitNext := 0
	for peerId, index := range n.nextIndex {
		termAtPeerIndex, err := n.log.GetTermAtIndex(index)
		if err != nil {
			logger.Fatalf("Failed to get term at index %d from log.", index)
			continue
		}
		peerResponse := n.network.GetAppendEntryResponse(index, termAtPeerIndex, peerId)
		switch peerResponse {
		case network.Success:
			n.matchIndex[peerId] = n.nextIndex[peerId]
			n.nextIndex[peerId] += 1
		case network.LogInconsistency:
			n.nextIndex[peerId] -= 1
		}

		if n.matchIndex[peerId] != n.log.GetLastIndex() {
			logEntry, err := n.log.GetEntry(index)
			if err != nil {
				logger.Fatalf("Failed to get entry at index %d from log.", index)
				continue
			}
			n.network.SendAppendEntryAsync(peerId, logEntry)
		}

		if n.commitIndex < n.log.GetLastIndex() {
			nextCommitLogEntry, err := n.log.GetEntry(n.commitIndex + 1)
			if err != nil {
				logger.Fatalf("Failed to get entry for next commit index %d from log.", n.commitIndex+1)
				continue
			}
			if n.matchIndex[peerId] > n.commitIndex && nextCommitLogEntry.Term == n.GetCurrentTerm() {
				numReadyToCommitNext += 1
			}
		}
	}

	if numReadyToCommitNext > len(n.matchIndex) {
		n.commitIndex += 1
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
			if networkGreatestTerm > n.GetCurrentTerm() {
				n.SetCurrentTerm(networkGreatestTerm)
				n.SetState(node.Follower)
			}

			switch n.GetState() {
			case node.Follower:
				n.runFollowerIteration()
			case node.Candidate:
				n.runCandidateIteration()
			case node.Leader:
				n.runLeaderIteration()
			}

			if n.commitIndex > n.lastApplied {
				logEntry, err := n.log.GetEntry(n.lastApplied + 1)
				if err != nil {
					logger.Fatalf("Failed to get next entry to apply at index %d from log.", n.lastApplied+1)
					continue
				}
				err = n.stateMachine.Apply(logEntry.Entry)
				if err != nil {
					logger.Fatalf("State machine unable to apply entry at index %d, term %d.", logEntry.Index, logEntry.Term)
				}
				n.lastApplied += 1
			}
		}
	}
}

func (n *Node) HandleReplicationRequest(ctx context.Context, clientID string, serializationID uint64, entry *any.Any) (node.ReplicationResponse, error) {
	res := node.ReplicationResponse{}
	if n.GetState() != node.Leader {
		res.ReplicationStatus = node.NotLeader
		// best effort, might not be leader
		res.LeaderID = n.GetVotedFor()
		return res, nil
	}

	n.log.AppendEntry(n.GetCurrentTerm(), clientID, serializationID, entry)

	select {
	case <-ctx.Done():
		return res, nil
	case result := <-n.stateMachine.WaitForResult(ctx, clientID, serializationID):
		if result.Error != nil {
			logger.Fatalf("State machine returned error for clientID %s, serializationID %d.", clientID, serializationID)
			res.ReplicationStatus = node.ApplyError
			return res, nil
		}
		res.Result = result.Result
		res.ReplicationStatus = node.Replicated
		return res, nil
	}
}

func (n *Node) HandleQueryRequest(ctx context.Context) {
	// TODO: implement query request handler
}

func (n *Node) GetLogLength() uint64 {
	return 0 // TODO: implement
}

func (n *Node) GetTermAtIndex(index uint64) (uint64, error) {
	return n.log.GetTermAtIndex(index)
}

func (n *Node) AppendEntry(entry *entries.LogEntry) error {
	return n.log.InsertLogEntry(entry)
}

func (n *Node) VotedForTerm(term uint64) bool {
	// TODO: implement voted for term
	return false
}

func (n *Node) GetLastLogIndex() uint64 {
	return n.log.GetLastIndex()
}
