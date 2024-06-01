package raft_network

import (
	"github.com/r-moraru/modular-raft/log"
	"github.com/r-moraru/modular-raft/node"
	"github.com/r-moraru/modular-raft/proto/raft_service"
)

type Network struct {
	nodeId string
	peers  map[string]raft_service.RaftServiceClient

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

// func (n *Network) SendRequestVote(term uint64) {

// }
