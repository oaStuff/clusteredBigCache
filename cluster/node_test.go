package cluster

import (
	"testing"
	"time"
)

func TestNodeConncting(t *testing.T)  {

	s := newTestServer(9093, true)
	err := s.start()
	if err != nil {
		panic(err)
	}
	defer s.close()

	node := NewNode(&NodeConfig{Join: true, LocalPort: 9999, ConnectRetries: 2}, nil)
	if err := node.Start(); err == nil {
		t.Error("node should not be able to start when 'join' is true and there is no joinIp")
		return
	}

	node.ShutDown()
	node = NewNode(&NodeConfig{Join: true, LocalPort: 9999, ConnectRetries: 2}, nil)
	node.config.JoinIp = "localhost:9093"
	if err = node.Start(); err != nil {
		t.Error(err)
		return
	}

	s.sendVerifyMessage("server1")
	time.Sleep(time.Second * 1)
	if node.remoteNodes.Size() != 1 {
		t.Error("only one node ought to be connected")
	}

	if _, ok := node.pendingConn.Load("remote_1"); !ok {
		t.Error("there should be a remote_1 server we are trying to connect to")
	}

}

func TestVerifyRemoteNode(t *testing.T)  {

	node := NewNode(&NodeConfig{Join: true, LocalPort: 9999, ConnectRetries: 2}, nil)
	rn := newRemoteNode(&remoteNodeConfig{IpAddress: "localhost:9092", Sync: false,
		PingFailureThreshHold: 1, PingInterval: 0}, node, nil)

	if !node.eventVerifyRemoteNode(rn) {
		t.Error("remoted node ought to be added")
	}

	if node.eventVerifyRemoteNode(rn) {
		t.Error("duplicated remote node(with same Id) should not be added twice")
	}

	node.eventRemoteNodeDisconneced(rn)

	if !node.eventVerifyRemoteNode(rn) {
		t.Error("remote node ought to be added after been removed")
	}

	node.ShutDown()
}

func TestBringingUpNode(t *testing.T)  {

	node := NewNode(&NodeConfig{Join: false, LocalPort: 1999, ConnectRetries: 2}, nil)
	if err := node.Start(); err != nil {
		t.Log(err)
		t.Error("node could not be brougth up")
	}
	node.ShutDown()
}
