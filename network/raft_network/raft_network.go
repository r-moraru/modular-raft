package raft_network

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/r-moraru/modular-raft/log"
	"github.com/r-moraru/modular-raft/network"
	"github.com/r-moraru/modular-raft/node"
	"github.com/r-moraru/modular-raft/proto/entries"
	"github.com/r-moraru/modular-raft/proto/raft_service"
)

type Network struct {
	nodeId          string
	peers           map[string]raft_service.RaftServiceClient
	electionTimeout uint64

	log  log.Log
	node node.Node
}

func (n *Network) GetId() string {
	return n.nodeId
}

func (n *Network) GetPeerList() []string {
	peerList := make([]string, len(n.peers))
	for peerId := range n.peers {
		peerList = append(peerList, peerId)
	}
	return peerList
}

type counter uint64

func (c *counter) incrementCounter() uint64 {
	return atomic.AddUint64((*uint64)(c), 1)
}

func (c *counter) getCounter() uint64 {
	return atomic.LoadUint64((*uint64)(c))
}

func (n *Network) SendSingleRequestVote(ctx context.Context, peerClient raft_service.RaftServiceClient, term uint64) chan bool {
	resChan := make(chan bool, 1)
	lastIndex := n.log.GetLastIndex()
	lastTerm, err := n.log.GetTermAtIndex(lastIndex)
	if err != nil {
		// TODO: log error
		resChan <- false
		return resChan
	}

	go func() {
		res, err := peerClient.RequestVote(ctx, &raft_service.RequestVoteRequest{
			Term:         term,
			CandidateId:  n.GetId(),
			LastLogIndex: lastIndex,
			LastLogTerm:  lastTerm,
		})
		if err != nil {
			resChan <- false
		}
		resChan <- res.VoteGranted
	}()

	return resChan
}

func (n *Network) SendRequestVote(ctx context.Context, term uint64) chan bool {
	majorityVoteChan := make(chan bool, 1)
	votesFor := counter(0)

	wg := sync.WaitGroup{}
	for _, peerClient := range n.peers {
		localPeerClient := peerClient
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case gotVote := <-n.SendSingleRequestVote(ctx, localPeerClient, term):
				if gotVote {
					votesFor.incrementCounter()
				}
			case <-ctx.Done():
			}
		}()
	}

	go func() {
		wg.Wait()
		if (votesFor.getCounter() + 1) >= uint64((len(n.peers)+1)/2) {
			majorityVoteChan <- true
		} else {
			majorityVoteChan <- false
		}
	}()

	return majorityVoteChan
}

func (n *Network) SendHeartbeat(ctx context.Context, peerId string) {
	n.SendAppendEntry(ctx, peerId, nil)
}

func (n *Network) SendAppendEntry(ctx context.Context, peerId string, entry *entries.LogEntry) network.ResponseStatus {
	peerClient, found := n.peers[peerId]
	if !found {
		// TODO: log error
		return network.NotReceived
	}
	lastIndex := n.log.GetLastIndex()
	lastTerm, err := n.log.GetTermAtIndex(lastIndex)
	if err != nil {
		// TODO: log error
		return network.NotReceived
	}

	resChan := make(chan *raft_service.AppendEntriesResponse, 1)
	go func() {
		res, _ := peerClient.AppendEntries(ctx, &raft_service.AppendEntriesRequest{
			Term:         n.node.GetCurrentTerm(),
			LeaderId:     n.GetId(),
			PrevLogIndex: lastIndex,
			PrevLogTerm:  lastTerm,
			LeaderCommit: n.node.GetCommitIndex(),
			Entry:        entry,
		})
		if err != nil {
			resChan <- nil
		}
		resChan <- res
	}()

	select {
	case res := <-resChan:
		if res == nil {
			return network.NotReceived
		}
		if res.Term > n.node.GetCurrentTerm() {
			return network.TermIssue
		}
		if res.Success {
			return network.Success
		} else {
			return network.LogInconsistency
		}
	case <-ctx.Done():
		return network.NotReceived
	}
}
