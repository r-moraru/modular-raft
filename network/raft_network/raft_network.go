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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Network struct {
	NodeId          string
	Peers           map[string]raft_service.RaftServiceClient
	ElectionTimeout uint64

	Log  log.Log
	Node node.Node
}

func BuildPeerMap(nodeNum int, nodeIds []string, nodeAddrs []string) map[string]raft_service.RaftServiceClient {
	peers := make(map[string]raft_service.RaftServiceClient)
	for i := 0; i < len(nodeIds); i++ {
		if i == nodeNum {
			continue
		}
		slog.Info("Adding peer with address " + nodeAddrs[i] + "\n")
		conn, err := grpc.Dial(
			nodeAddrs[i],
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		slog.Info("Successfully added peer " + nodeIds[i] + " with address " + nodeAddrs[i] + "\n")
		if err != nil {
			slog.Error("Error connecting peer" + nodeAddrs[i] + "\n")
		}
		c := raft_service.NewRaftServiceClient(conn)
		peers[nodeIds[i]] = c
	}
	return peers
}

func (n *Network) Init(raftNode node.Node, log log.Log, nodeNum int, nodeIds []string, nodeAddrs []string) {
	n.Node = raftNode
	n.Log = log
	n.NodeId = nodeIds[nodeNum]
	n.Peers = BuildPeerMap(nodeNum, nodeIds, nodeAddrs)
}

func (n *Network) GetId() string {
	return n.NodeId
}

func (n *Network) GetPeerList() []string {
	peerList := make([]string, 0)
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
	lastIndex := n.Log.GetLastIndex()
	lastTerm, err := log.GetTermAtIndexHelper(n.Log, lastIndex)
	if err != nil {
		slog.Error("SEND REQUEST VOTE - Error fetching term at PrevLogIndex.")
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
			slog.Error(err.Error())
			resChan <- false
		} else {
			slog.Info(fmt.Sprintf("SEND REQUEST VOTE - Vote granted from peer: %v\n", res.VoteGranted))
			resChan <- res.VoteGranted
		}
	}()

	return resChan
}

func (n *Network) SendRequestVote(ctx context.Context, term uint64) chan bool {
	majorityVoteChan := make(chan bool, 1)
	votesFor := counter(0)

	wg := sync.WaitGroup{}
	for _, peerClient := range n.Peers {
		if n.Node.GetState() != node.Candidate {
			continue
		}
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
				slog.Error("SEND REQUEST VOTE - Request to peer timed out.\n")
			}
		}()
	}

	go func() {
		wg.Wait()
		if (votesFor.getCounter() + 1) > uint64((len(n.Peers)+1)/2) {
			majorityVoteChan <- true
		} else {
			majorityVoteChan <- false
		}
	}()

	return majorityVoteChan
}

func (n *Network) SendAppendEntryHandler(ctx context.Context, peerId string, prevIndex uint64, prevTerm uint64, entry *entries.LogEntry) network.ResponseStatus {
	peerClient, found := n.Peers[peerId]
	if !found {
		slog.Error("SEND APPEND ENTRY HANDLER - Cannot find peer client.")
		return network.NotReceived
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
			slog.Error("SEND APPEND ENTRY HANDLER - AppendEntry was not received.")
			return network.NotReceived
		}
		if res.Term > n.Node.GetCurrentTerm() {
			slog.Error(fmt.Sprintf("SEND APPEND ENTRY HANDLER - TERM ISSUE: peer term: %d, local term: %d\n", res.Term, n.Node.GetCurrentTerm()))
			return network.TermIssue
		}
		if res.Success {
			return network.Success
		} else {
			return network.LogInconsistency
		}
	case <-ctx.Done():
		slog.Error("SEND APPEND ENTRY HANDLER - AppendEntry request not received by " + peerId + "\n")
		return network.NotReceived
	}
}

func (n *Network) SendHeartbeat(ctx context.Context, peerId string, prevIndex uint64) network.ResponseStatus {
	prevTerm, err := log.GetTermAtIndexHelper(n.Log, prevIndex)
	if err != nil {
		slog.Error("SEND HEARTBEAT - Append entry not received by " + peerId + "\n")
		return network.NotReceived
	}
	return n.SendAppendEntryHandler(ctx, peerId, prevIndex, prevTerm, nil)
}

func (n *Network) SendAppendEntry(ctx context.Context, peerId string, entry *entries.LogEntry) network.ResponseStatus {
	prevIndex := entry.Index - 1
	prevTerm, err := log.GetTermAtIndexHelper(n.Log, prevIndex)
	if err != nil {
		slog.Error("SEND APPEND ENTRY - Append entry not received by " + peerId + "\n")
		return network.NotReceived
	}
	return n.SendAppendEntryHandler(ctx, peerId, prevIndex, prevTerm, entry)
}
