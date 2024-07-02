package raft_node

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/r-moraru/modular-raft/log"
	"github.com/r-moraru/modular-raft/network"
	"github.com/r-moraru/modular-raft/node"
	"github.com/r-moraru/modular-raft/proto/entries"
	"github.com/r-moraru/modular-raft/state_machine"
)

type Node struct {
	state     node.State
	stateLock *sync.RWMutex

	timer           *time.Timer
	timerMutex      *sync.RWMutex
	electionTimeout uint64
	heartbeat       uint64

	// Election state
	votedFor            *string
	votedForLock        *sync.RWMutex
	currentLeaderID     string
	currentLeaderIDLock *sync.RWMutex

	currentTerm     uint64
	currentTermLock *sync.RWMutex
	commitIndex     uint64
	commitIndexLock *sync.RWMutex
	lastApplied     uint64

	// Leader bookkeeping
	nextIndex  sync.Map
	matchIndex sync.Map

	log          log.Log
	stateMachine state_machine.StateMachine
	network      network.Network
}

// TODO: make sure network always has odd number of members
func New(electionTimeout uint64, heartbeat uint64, log log.Log, stateMachine state_machine.StateMachine, network network.Network) (*Node, error) {
	var currentTerm uint64
	var err error
	if log.GetLength() == 0 {
		currentTerm = 0
	} else {
		currentTerm, err = log.GetTermAtIndex(log.GetLastIndex())
		if err != nil {
			return nil, err
		}
	}
	return &Node{
		state:               node.Follower,
		stateLock:           new(sync.RWMutex),
		timer:               time.NewTimer(time.Duration(0)),
		timerMutex:          &sync.RWMutex{},
		electionTimeout:     electionTimeout,
		heartbeat:           heartbeat,
		currentTerm:         currentTerm,
		currentTermLock:     &sync.RWMutex{},
		votedFor:            nil,
		votedForLock:        &sync.RWMutex{},
		currentLeaderID:     network.GetId(),
		currentLeaderIDLock: &sync.RWMutex{},
		commitIndex:         stateMachine.GetLastApplied(),
		commitIndexLock:     &sync.RWMutex{},
		lastApplied:         stateMachine.GetLastApplied(),
		nextIndex:           sync.Map{},
		matchIndex:          sync.Map{},
		log:                 log,
		stateMachine:        stateMachine,
		network:             network,
	}, nil
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

func (n *Node) VotedForTerm() bool {
	n.votedForLock.RLock()
	defer n.votedForLock.RUnlock()
	return n.votedFor != nil
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

func (n *Node) GetCommitIndex() uint64 {
	n.commitIndexLock.RLock()
	defer n.commitIndexLock.RUnlock()
	return n.commitIndex
}

func (n *Node) SetCommitIndex(commitIndex uint64) {
	n.commitIndexLock.Lock()
	defer n.commitIndexLock.Unlock()
	n.commitIndex = commitIndex
}

func (n *Node) ResetTimer() {
	n.timerMutex.Lock()
	defer n.timerMutex.Unlock()
	n.timer = time.NewTimer(time.Duration(n.electionTimeout+uint64(rand.Intn(int(n.electionTimeout)/2))) * time.Millisecond)
}

func (n *Node) runFollowerIteration() {
	n.timerMutex.RLock()
	defer n.timerMutex.RUnlock()
	select {
	case <-n.timer.C:
		slog.Info("ELECTION TIMEOUT - turning into candidate.")
		n.SetState(node.Candidate)
	default:
		return
	}
}

func (n *Node) resetLeaderBookkeeping() {
	lastIndex := n.log.GetLastIndex()
	for _, peerId := range n.network.GetPeerList() {
		n.matchIndex.Store(peerId, uint64(0))
		n.nextIndex.Store(peerId, lastIndex+1)
	}
}

func (n *Node) runCandidateIteration() {
	n.SetCurrentTerm(n.GetCurrentTerm() + 1)
	n.SetVotedFor(n.network.GetId())
	n.ResetTimer()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	select {
	// TODO(Optional): also check network for received AppendEntriesRPC from new leader
	case gotVoted := <-n.network.SendRequestVote(ctx, n.GetCurrentTerm()):
		n.ClearVotedFor()
		if gotVoted {
			slog.Info("State changed to LEADER.\n")
			n.SetState(node.Leader)
			n.SetCurrentLeaderID(n.network.GetId())
			n.resetLeaderBookkeeping()
			// TODO: commit blank no-op to prevent stale reads
		} else {
			n.SetState(node.Follower)
		}
	case <-n.timer.C:
		return
	}
}

func (n *Node) runLeaderIteration() {
	heartbeatTimer := time.NewTimer(time.Duration(n.heartbeat) * time.Millisecond)
	defer func() {
		<-heartbeatTimer.C
	}()

	<-n.SendAppendEntriesToPeers()

	sortedMatchIndexes := []uint64{n.log.GetLastIndex()}
	n.matchIndex.Range(func(key any, value any) bool {
		matchIndex := value.(uint64)
		sortedMatchIndexes = append(sortedMatchIndexes, matchIndex)
		return true
	})
	sort.Slice(sortedMatchIndexes, func(i, j int) bool {
		return sortedMatchIndexes[i] < sortedMatchIndexes[j]
	})
	majorityMatchIndex := sortedMatchIndexes[len(sortedMatchIndexes)/2]
	nextCommitIndex := max(majorityMatchIndex, n.GetCommitIndex()+1)
	termAtNextCommitIndex, err := n.log.GetTermAtIndex(nextCommitIndex)
	if err == nil && termAtNextCommitIndex == n.GetCurrentTerm() {
		n.SetCommitIndex(nextCommitIndex)
	}
}

func (n *Node) runIteration() {
	switch n.GetState() {
	case node.Follower:
		n.runFollowerIteration()
	case node.Candidate:
		n.runCandidateIteration()
	case node.Leader:
		n.runLeaderIteration()
	}

	if n.commitIndex > n.lastApplied && n.commitIndex <= n.log.GetLastIndex() {
		logEntry, err := n.log.GetEntry(n.lastApplied + 1)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to get next entry to apply at index %d from log.", n.lastApplied+1))
			return
		}
		err = n.stateMachine.Apply(logEntry)
		if err != nil {
			slog.Error(fmt.Sprintf("State machine unable to apply entry at index %d, term %d.", logEntry.Index, logEntry.Term))
		} else {
			n.lastApplied += 1
		}
	}
}

func (n *Node) Run(ctx context.Context) {
	n.ResetTimer()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n.runIteration()
		}
	}
}

func (n *Node) SendAppendEntry(peerId string, logEntry *entries.LogEntry, peerNextIndex uint64, peerMatchIndex uint64) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(n.heartbeat)*time.Millisecond)
	defer cancel()
	responseStatus := n.network.SendAppendEntry(ctx, peerId, logEntry)

	switch responseStatus {
	case network.Success:
		n.matchIndex.Store(peerId, peerNextIndex)
		n.nextIndex.Store(peerId, peerNextIndex+1)
	case network.LogInconsistency:
		n.nextIndex.Store(peerId, peerNextIndex-1)
		if peerMatchIndex >= peerNextIndex {
			n.matchIndex.Store(peerId, peerNextIndex-1)
		}
	}
}

func (n *Node) SendAppendEntriesToPeers() chan struct{} {
	wg := sync.WaitGroup{}

	n.nextIndex.Range(func(key any, value any) bool {
		peerId := key.(string)
		peerNextIndex := value.(uint64)
		peerMatchIndexVal, found := n.matchIndex.Load(peerId)
		peerMatchIndex := peerMatchIndexVal.(uint64)
		if !found {
			slog.Error(fmt.Sprintf("Leader bookkeeping mismatch: peer %s from nextIndex not found in matchIndex.", peerId))
			return true
		}

		slog.Info(fmt.Sprintf("PEER MATCH INDEX FOR PEER %s: %d", peerId, peerMatchIndex))
		slog.Info(fmt.Sprintf("PEER NEXT INDEX FOR PEER %s: %d", peerId, peerNextIndex))
		if peerMatchIndex != n.log.GetLastIndex() && peerNextIndex <= n.log.GetLastIndex() {

			logEntry, err := n.log.GetEntry(peerNextIndex)
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to get entry at index %d from log.", peerNextIndex))
				return true
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				slog.Info("Sending append entry to " + peerId + "\n")
				n.SendAppendEntry(peerId, logEntry, peerNextIndex, peerMatchIndex)
			}()
		} else {
			slog.Info("Sending heartbeat to " + peerId + "\n")
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(n.heartbeat)*time.Millisecond)
				defer cancel()
				responseStatus := n.network.SendHeartbeat(ctx, peerId, peerNextIndex-1)

				if responseStatus == network.LogInconsistency {
					n.nextIndex.Store(peerId, peerNextIndex-1)
					if peerMatchIndex >= peerNextIndex {
						n.matchIndex.Store(peerId, peerNextIndex-1)
					}
				}
			}()
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
