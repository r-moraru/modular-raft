package raft_node

import (
	"context"
	logger "log"
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
	currentTerm, err := log.GetTermAtIndex(log.GetLastIndex())
	if err != nil {
		return nil, err
	}
	return &Node{
		state:               node.Follower,
		stateLock:           &sync.RWMutex{},
		timer:               time.NewTimer(time.Duration(0)),
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
	// TODO: randomize timer around electionTimeout range
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

func (n *Node) resetLeaderBookkeeping() {
	lastIndex := n.log.GetLastIndex()
	for _, peerId := range n.network.GetPeerList() {
		n.matchIndex.Store(peerId, 0)
		n.nextIndex.Store(peerId, lastIndex)
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

	if n.commitIndex > n.lastApplied {
		logEntry, err := n.log.GetEntry(n.lastApplied + 1)
		if err != nil {
			logger.Fatalf("Failed to get next entry to apply at index %d from log.", n.lastApplied+1)
			return
		}
		err = n.stateMachine.Apply(logEntry)
		if err != nil {
			logger.Fatalf("State machine unable to apply entry at index %d, term %d.", logEntry.Index, logEntry.Term)
		}
		n.lastApplied += 1
	}
}

func (n *Node) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n.runIteration()
		}
	}
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
