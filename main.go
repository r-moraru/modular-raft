package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/r-moraru/modular-raft/examples/kv_store"
	"github.com/r-moraru/modular-raft/raft_server"
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

	log := &kv_store.InMemoryLog{
		Mutex: &sync.Mutex{},
	}
	stateMachine := &kv_store.KvStore{}

	raftServer, err := raft_server.New(log, stateMachine, 5000, 1000)
	if err != nil {
		slog.Error("Failed to initialize raft server.")
		return
	}
	raftServer.Start(*nodeNum, nodeIds, nodeAddrs)

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
