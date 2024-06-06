package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"

	"github.com/r-moraru/modular-raft/examples/kv_store"
	"github.com/r-moraru/modular-raft/network/raft_network"
	"github.com/r-moraru/modular-raft/node/raft_node"
	"github.com/r-moraru/modular-raft/proto/raft_service"
	"github.com/r-moraru/modular-raft/raft_server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	nodeNum := flag.Int("node_num", 0, "")
	// startAsLeader := flag.Bool("start_as_leader", false, "")

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
	network := &raft_network.Network{}
	raftNode, _ := raft_node.New(5000, 1000, log, stateMachine, network)
	raftService := &raft_network.RaftService{
		RaftNode: raftNode,
		Log:      log,
	}

	raftServer := raft_server.RaftServer{
		Log:          log,
		Node:         raftNode,
		StateMachine: stateMachine,
	}

	slog.Info("Starting raft consensus service at " + nodeAddrs[*nodeNum])
	lis, err := net.Listen("tcp", nodeAddrs[*nodeNum])
	if err != nil {
		fmt.Println("failed" + err.Error())
		return
	}
	s := grpc.NewServer()
	raft_service.RegisterRaftServiceServer(s, raftService)

	go func() {
		if err := s.Serve(lis); err != nil {
			fmt.Println("failed serve")
		}
	}()

	network.Node = raftNode
	network.Log = log
	network.NodeId = nodeIds[*nodeNum]
	network.Peers = make(map[string]raft_service.RaftServiceClient)
	var wg sync.WaitGroup
	for i := 0; i < len(nodeIds); i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i == *nodeNum {
				return
			}
			slog.Info("Adding peer with address " + nodeAddrs[i] + "\n")
			conn, err := grpc.Dial(
				nodeAddrs[i],
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithBlock(),
			)
			slog.Info("Successfully added peer " + nodeIds[i] + " with address " + nodeAddrs[i] + "\n")
			if err != nil {
				slog.Error("big time error connecting peer" + nodeAddrs[i] + "\n")
			}
			c := raft_service.NewRaftServiceClient(conn)
			network.Peers[nodeIds[i]] = c
		}(i)
	}

	wg.Wait()

	go func() {
		raftNode.Run(context.Background())
	}()

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
