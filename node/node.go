package node

import (
	"context"
	logger "log"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/any"
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

type ReplicationStatus int64

const (
	NotLeader ReplicationStatus = iota
	NotTracked
	InProgress
	Replicated
	ApplyError
)

type Node struct {
	state           State
	stateLock       *sync.RWMutex
	timer           time.Timer
	electionTimeout int64
	heartbeat       int64

	currentTerm     int64
	currentTermLock *sync.RWMutex
	votedFor        network.PeerId
	votedForLock    *sync.RWMutex
	commitIndex     int64
	lastApplied     int64
	nextIndex       map[network.PeerId]int64
	matchIndex      map[network.PeerId]int64

	log          log.Log
	stateMachine state_machine.StateMachine
	network      network.Network
}

type ReplicationResponse struct {
	ReplicationStatus ReplicationStatus
	LeaderID          network.PeerId
	Result            *any.Any
}

func (n *Node) getState() State {
	n.stateLock.RLock()
	defer n.stateLock.RUnlock()
	return n.state
}

func (n *Node) setState(newState State) {
	n.stateLock.Lock()
	defer n.stateLock.Unlock()
	n.state = newState
}

func (n *Node) getVotedFor() network.PeerId {
	n.votedForLock.RLock()
	defer n.votedForLock.RUnlock()
	return n.votedFor
}

func (n *Node) setVotedFor(peerId network.PeerId) {
	n.votedForLock.Lock()
	defer n.votedForLock.Unlock()
	n.votedFor = peerId
}

func (n *Node) getCurrentTerm() int64 {
	n.currentTermLock.RLock()
	defer n.currentTermLock.RUnlock()
	return n.currentTerm
}

func (n *Node) setCurrentTerm(newTerm int64) {
	n.currentTermLock.Lock()
	defer n.currentTermLock.Unlock()
	n.currentTerm = newTerm
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
		n.setState(Candidate)
	default:
		return
	}
}

func (n *Node) runCandidateIteration() {
	n.setCurrentTerm(n.getCurrentTerm() + 1)
	n.setVotedFor(n.network.GetId())
	n.resetTimer()
	n.network.SendRequestVoteAsync(n.getCurrentTerm())
	select {
	// TODO(Optional): also check network for received AppendEntriesRPC from new leader
	case <-n.network.GotMajorityVote():
		n.setState(Leader)
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
			if n.matchIndex[peerId] > n.commitIndex && nextCommitLogEntry.Term == n.getCurrentTerm() {
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
			if networkGreatestTerm > n.getCurrentTerm() {
				n.setCurrentTerm(networkGreatestTerm)
				n.setState(Follower)
			}

			switch n.getState() {
			case Follower:
				n.runFollowerIteration()
			case Candidate:
				n.runCandidateIteration()
			case Leader:
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

func (n *Node) HandleReplicationRequest(ctx context.Context, clientID string, serializationID int64, entry *any.Any) (ReplicationResponse, error) {
	// TODO: implement setters and getters with read write mutex for state, currentTerm, votedFor
	res := ReplicationResponse{}
	if n.getState() != Leader {
		res.ReplicationStatus = NotLeader
		// best effort, might not be leader
		res.LeaderID = n.getVotedFor()
		return res, nil
	}

	n.log.AppendEntry(n.getCurrentTerm(), clientID, serializationID, entry)

	select {
	case <-ctx.Done():
		return res, nil
	case result := <-n.stateMachine.WaitForResult(ctx, clientID, serializationID):
		if result.Error != nil {
			logger.Fatalf("State machine returned error for clientID %s, serializationID %d.", clientID, serializationID)
			res.ReplicationStatus = ApplyError
			return res, nil
		}
		res.Result = result.Result
		res.ReplicationStatus = Replicated
		return res, nil
	}
}

func (n *Node) HandleQueryRequest(ctx context.Context) {

}
