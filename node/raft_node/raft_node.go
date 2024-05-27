package raft_node

import (
	"context"
	logger "log"
	"sync"
	"time"

	anypb "github.com/golang/protobuf/ptypes/any"
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

	currentTerm         uint64
	currentTermLock     *sync.RWMutex
	votedFor            *string
	votedForLock        *sync.RWMutex
	currentLeaderID     string
	currentLeaderIDLock *sync.RWMutex
	commitIndex         uint64
	lastApplied         uint64
	nextIndex           sync.Map
	matchIndex          sync.Map

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

	if n.state == node.Follower {
		n.SetCurrentLeaderID(n.GetVotedFor())
		n.ClearVotedFor()
	}
}

func (n *Node) GetVotedFor() string {
	n.votedForLock.RLock()
	defer n.votedForLock.RUnlock()
	return *n.votedFor
}

func (n *Node) ClearVotedFor() {
	n.votedForLock.Lock()
	defer n.votedForLock.Unlock()
	n.votedFor = nil
}

func (n *Node) SetVotedFor(peerId string) {
	n.votedForLock.Lock()
	defer n.votedForLock.Unlock()
	n.votedFor = &peerId
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

func (n *Node) GetCurrentLeaderID() string {
	n.currentLeaderIDLock.RLock()
	defer n.currentLeaderIDLock.RUnlock()
	return n.currentLeaderID
}

func (n *Node) SetCurrentLeaderID(leaderID string) {
	n.currentLeaderIDLock.Lock()
	defer n.currentLeaderIDLock.Unlock()
	n.currentLeaderID = leaderID
}

func (n *Node) ResetTimer() {
	n.timer.Reset(time.Duration(n.electionTimeout) * time.Millisecond)
}

func (n *Node) runFollowerIteration() {
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
	n.ResetTimer()
	n.network.SendRequestVoteAsync(n.GetCurrentTerm())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	select {
	// TODO(Optional): also check network for received AppendEntriesRPC from new leader
	case <-n.network.GotMajorityVote(ctx):
		n.SetState(node.Leader)
		// TODO: commit blank no-op to prevent stale reads
		// TODO: reset leader bookkeeping
	case <-n.timer.C:
		return
	}
}

func (n *Node) runLeaderIteration() {
	heartbeatTimer := time.NewTimer(time.Duration(n.heartbeat) * time.Millisecond)

	<-n.SendAppendEntriesToPeers()

	numReadyToCommitNext := 0
	numPeers := 0
	n.matchIndex.Range(func(key any, value any) bool {
		numPeers += 1
		peerMatchIndex := value.(uint64)
		if n.commitIndex < n.log.GetLastIndex() {
			nextCommitIndexTerm, err := n.log.GetTermAtIndex(n.commitIndex + 1)
			if err != nil {
				logger.Fatalf("Failed to get entry for next commit index %d from log.", n.commitIndex+1)
				return true
			}
			if peerMatchIndex > n.commitIndex && nextCommitIndexTerm == n.GetCurrentTerm() {
				numReadyToCommitNext += 1
			}
		}
		return true
	})

	if numReadyToCommitNext > numPeers/2 {
		n.commitIndex += 1
	}

	<-heartbeatTimer.C
}

func (n *Node) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
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

func (n *Node) HandleReplicationRequest(ctx context.Context, clientID string, serializationID uint64, entry *anypb.Any) (node.ReplicationResponse, error) {
	res := node.ReplicationResponse{}
	if n.GetState() != node.Leader {
		res.ReplicationStatus = node.NotLeader
		// best effort, might not be leader
		res.LeaderID = n.GetCurrentLeaderID()
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
	return n.log.GetLength()
}

func (n *Node) GetTermAtIndex(index uint64) (uint64, error) {
	return n.log.GetTermAtIndex(index)
}

func (n *Node) AppendEntry(entry *entries.LogEntry) error {
	return n.log.InsertLogEntry(entry)
}

func (n *Node) GetLastLogIndex() uint64 {
	return n.log.GetLastIndex()
}

func (n *Node) SendAppendEntry(peerId string, logEntry *entries.LogEntry) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(n.heartbeat)*time.Millisecond)
	defer cancel()
	responseStatus := n.network.SendAppendEntry(ctx, peerId, logEntry)
	value, loaded := n.nextIndex.Load(peerId)
	if !loaded {
		logger.Fatalf("Bookkeeping error - peer %s not found.", peerId)
		return
	}
	nextIndex, ok := value.(uint64)
	if !ok {
		logger.Fatal("Bookkeeping error - next index value is not uint64.")
	}
	switch responseStatus {
	case network.Success:
		n.matchIndex.Store(peerId, nextIndex)
		n.nextIndex.Store(peerId, nextIndex+1)
	case network.LogInconsistency:
		n.nextIndex.Store(peerId, nextIndex-1)
	}
}

func (n *Node) SendAppendEntriesToPeers() chan struct{} {
	wg := sync.WaitGroup{}

	n.nextIndex.Range(func(key any, value any) bool {
		peerId := key.(string)
		peerNextIndex := value.(uint64)
		peerMatchIndex, found := n.matchIndex.Load(peerId)
		if !found {
			logger.Fatalf("Leader bookkeeping mismatch: peer %s from nextIndex not found in matchIndex.", peerId)
			return true
		}
		if peerMatchIndex != n.log.GetLastIndex() {
			logEntry, err := n.log.GetEntry(peerNextIndex)
			if err != nil {
				logger.Fatalf("Failed to get entry at index %d from log.", peerNextIndex)
				return true
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				n.SendAppendEntry(peerId, logEntry)
			}()
		} else {
			n.network.SendHeartbeat(peerId)
		}

		return true
	})

	done := make(chan struct{}, 1)
	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	return done
}
