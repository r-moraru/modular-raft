package raft_network

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/r-moraru/modular-raft/log"
	"github.com/r-moraru/modular-raft/network"
	"github.com/r-moraru/modular-raft/node"
	"github.com/r-moraru/modular-raft/proto/entries"
	"github.com/r-moraru/modular-raft/proto/raft_service"
)

type Network struct {
	NodeId          string
	Peers           map[string]raft_service.RaftServiceClient
	ElectionTimeout uint64

	Log  log.Log
	Node node.Node
}

func (n *Network) GetId() string {
	return n.NodeId
}

func (n *Network) GetPeerList() []string {
	peerList := make([]string, len(n.Peers))
	for peerId := range n.Peers {
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
	var lastTerm uint64
	var err error
	lastIndex := n.Log.GetLastIndex()
	if lastIndex == 0 {
		lastTerm = 0
	} else {
		lastTerm, err = n.Log.GetTermAtIndex(lastIndex)
		if err != nil {
			// TODO: log error
			resChan <- false
			return resChan
		}
	}

	go func() {
		res, err := peerClient.RequestVote(ctx, &raft_service.RequestVoteRequest{
			Term:         term,
			CandidateId:  n.GetId(),
			LastLogIndex: lastIndex,
			LastLogTerm:  lastTerm,
		})
		if err != nil {
			slog.Error(err.Error())
			resChan <- false
		} else {
			slog.Info(fmt.Sprintf("Vote granted from peer: %v\n", res.VoteGranted))
			resChan <- res.VoteGranted
		}
	}()

	return resChan
}

func (n *Network) SendRequestVote(ctx context.Context, term uint64) chan bool {
	majorityVoteChan := make(chan bool, 1)
	votesFor := counter(0)

	wg := sync.WaitGroup{}
	for peerId, peerClient := range n.Peers {
		if n.Node.GetState() != node.Candidate {
			continue
		}
		localPeerClient := peerClient
		localPeerId := peerId
		wg.Add(1)
		go func() {
			defer wg.Done()
			slog.Debug(fmt.Sprintf("Sending request to %s\n", localPeerId))
			select {
			case gotVote := <-n.SendSingleRequestVote(ctx, localPeerClient, term):
				if gotVote {
					votesFor.incrementCounter()
				}
			case <-ctx.Done():
				slog.Debug("context cancelled.\n")
			}
		}()
	}

	go func() {
		wg.Wait()
		slog.Info(fmt.Sprintf("Num votes: %d\n", votesFor.getCounter()+1))
		slog.Info(fmt.Sprintf("Minimum to get voted: %d\n", uint64((len(n.Peers)+1)/2)))
		if (votesFor.getCounter() + 1) > uint64((len(n.Peers)+1)/2) {
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
	peerClient, found := n.Peers[peerId]
	if !found {
		// TODO: log error
		return network.NotReceived
	}
	var prevIndex uint64
	var prevTerm uint64
	var err error
	if entry != nil {
		prevIndex = entry.Index - 1
		if prevIndex == 0 {
			prevTerm = 0
		} else {
			prevTerm, err = n.Log.GetTermAtIndex(prevIndex)
			if err != nil {
				// TODO: log error
				return network.NotReceived
			}
		}
	}

	resChan := make(chan *raft_service.AppendEntriesResponse, 1)
	go func() {
		res, err := peerClient.AppendEntries(ctx, &raft_service.AppendEntriesRequest{
			Term:         n.Node.GetCurrentTerm(),
			LeaderId:     n.GetId(),
			PrevLogIndex: prevIndex,
			PrevLogTerm:  prevTerm,
			LeaderCommit: n.Node.GetCommitIndex(),
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
		if res.Term > n.Node.GetCurrentTerm() {
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
