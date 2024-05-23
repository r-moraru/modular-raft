package node

import (
	"github.com/r-moraru/modular-raft/log"
	"github.com/r-moraru/modular-raft/network"
	"github.com/r-moraru/modular-raft/state_machine"
)

type State int

const (
	Leader State = iota
	Follower
	Candidate
)

type PeerId string

type Node struct {
	state State

	currentTerm int
	votedFor    PeerId
	commitIndex int
	lastApplied int
	nextIndex   map[PeerId]int
	matchIndex  map[PeerId]int

	log          log.Log
	stateMachine state_machine.StateMachine
	network      network.Network
}
