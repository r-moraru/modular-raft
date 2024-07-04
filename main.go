package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/r-moraru/modular-raft/clients"
	"github.com/r-moraru/modular-raft/raft_server"
	"github.com/r-moraru/modular-raft/state_machine_1"
	"github.com/r-moraru/modular-raft/test_implementations/kv_store"
)

func main() {
	nodeNum := flag.Int("node_num", 0, "")
	flag.Parse()
	nodeIds := []string{
		"node1", "node2", "node3",
	}
	nodeAddrs := []string{
		"localhost:5001",
		"localhost:5002",
		"localhost:5003",
	}
	listenAddrs := []string{
		"localhost:8081",
		"localhost:8082",
		"localhost:8083",
	}
	stateMachineAddrs := []string{
		"localhost:8090",
		"localhost:8091",
		"localhost:8092",
	}

	log := &kv_store.InMemoryLog{
		Mutex: &sync.Mutex{},
	}
	stateMachine := state_machine_1.New()
	go func() {
		stateMachine.Run(context.Background(), stateMachineAddrs[*nodeNum])
	}()

	stateMachineClient := clients.NewStateMachineClient(stateMachineAddrs[*nodeNum], 1000)

	raftServer, err := raft_server.New(log, stateMachineClient, 5000, 1000)
	if err != nil {
		slog.Error("Failed to initialize raft server.")
		return
	}
	raftServer.Start(*nodeNum, nodeIds, nodeAddrs, listenAddrs[*nodeNum])

	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		line := input.Text()
		response, err := raftServer.HandleReplicationRequest(context.Background(), "", 0, line)
		if err != nil {
			fmt.Printf("error %v\n", err)
		} else {
			fmt.Printf("response leader id: %s\n", response.LeaderID)
			fmt.Printf("result: %s\n", response.Result)
		}
	}

}
