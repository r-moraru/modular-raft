package raft_network

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

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

func (n *Network) SendRequestVote(term uint64) chan bool {
	majorityVoteChan := make(chan bool, 1)
	votesFor := counter(0)

	wg := sync.WaitGroup{}
	for _, peerClient := range n.peers {
		localPeerClient := peerClient

		wg.Add(1)
		go func() {
			defer wg.Done()
			lastIndex := n.log.GetLastIndex()
			lastTerm, err := n.log.GetTermAtIndex(lastIndex)
			if err != nil {
				// TODO: log error
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(n.electionTimeout)*time.Millisecond)
			defer cancel()
			res, err := localPeerClient.RequestVote(ctx, &raft_service.RequestVoteRequest{
				Term:         term,
				CandidateId:  n.GetId(),
				LastLogIndex: lastIndex,
				LastLogTerm:  lastTerm,
			})
			if err != nil {
				// TODO: log error
				return
			}
			if res.VoteGranted {
				votesFor.incrementCounter()
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
		resChan <- res
	}()

	select {
	case res := <-resChan:
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
