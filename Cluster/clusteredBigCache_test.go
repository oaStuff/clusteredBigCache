package clusteredBigCache

import (
	"testing"
	"time"
	"github.com/oaStuff/clusteredBigCache/utils"
)

func TestNodeConnecting(t *testing.T)  {

	s := utils.NewTestServer(9093, true)
	err := s.Start()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	node := New(&ClusteredBigCacheConfig{Join: true, LocalPort: 9998, ConnectRetries: 2}, nil)
	if err := node.Start(); err == nil {
		t.Error("node should not be able to start when 'join' is true and there is no joinIp")
		return
	}

	node.ShutDown()
	node = New(&ClusteredBigCacheConfig{Join: true, LocalPort: 9998, ConnectRetries: 2}, nil)
	node.config.JoinIp = "localhost:9093"
	if err = node.Start(); err != nil {
		t.Error(err)
		return
	}

	s.SendVerifyMessage("server1")
	time.Sleep(time.Second * 3)
	if node.remoteNodes.Size() != 1 {
		t.Log(node.remoteNodes.Size())
		t.Error("only one node ought to be connected")
	}

	if _, ok := node.pendingConn.Load("remote_1"); !ok {
		t.Error("there should be a remote_1 server we are trying to connect to")
	}

}

func TestVerifyRemoteNode(t *testing.T)  {

	node := New(&ClusteredBigCacheConfig{Join: true, LocalPort: 9999, ConnectRetries: 2}, nil)
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

	node := New(&ClusteredBigCacheConfig{Join: false, LocalPort: 1799, ConnectRetries: 2}, nil)
	if err := node.Start(); err != nil {
		t.Log(err)
		t.Error("node could not be brougth up")
	}
	node.ShutDown()
}

func TestPutData(t *testing.T)  {
	node1 := New(&ClusteredBigCacheConfig{Join: false, LocalPort: 1989, ConnectRetries: 2}, nil)
	node2 := New(&ClusteredBigCacheConfig{Join: true, LocalPort: 1998, JoinIp:"localhost:1989", ConnectRetries: 2}, nil)


	node1.Start()
	node2.Start()

	node1.Put("key_1", []byte("data_1"), time.Minute * 1)
	time.Sleep(time.Millisecond * 200)
	result, err := node2.Get("key_1", time.Millisecond * 200)
	if err != nil {
		t.Error(err)
	}

	if string(result) != "data_1" {
		t.Error("data placed in node1 not the same gotten from node2")
	}

	node2.Delete("key_1")
	time.Sleep(time.Millisecond * 200)
	result, err = node1.Get("key_1", time.Millisecond * 200)
	if err == nil {
		t.Error("error ought to be not found because the key and its data has been deleted")
	}

	node1.ShutDown()
	node2.ShutDown()
}

func TestPutDataWithPassiveClient(t *testing.T)  {
	node1 := New(&ClusteredBigCacheConfig{Join: false, LocalPort: 1979, ConnectRetries: 2}, nil)
	node2 := NewPassiveClient("localhost:1979", 1898, 5, 3, 10, nil)


	node1.Start()
	node2.Start()

	node1.Put("key_1", []byte("data_1"), time.Minute * 1)
	time.Sleep(time.Millisecond * 200)
	result, err := node2.Get("key_1", time.Millisecond * 200)

	if err != nil {
		t.Error(err)
	}

	if string(result) != "data_1" {
		t.Error("data placed in node1 not the same gotten from node2")
	}

	node2.Delete("key_1")
	time.Sleep(time.Millisecond * 200)
	result, err = node1.Get("key_1", time.Millisecond * 200)
	if err == nil {
		t.Error("error ought to be found because the key and its data has been deleted")
	}

	node1.ShutDown()
	node2.ShutDown()
}

func TestPassiveMode(t *testing.T)  {

	node1 := New(&ClusteredBigCacheConfig{Join: false, LocalPort: 1959, ConnectRetries: 2}, nil)

	client1 := NewPassiveClient("localhost:1959", 1897, 5, 3, 10,nil)
	client2 := NewPassiveClient("localhost:1897", 1996, 5, 3, 10,nil)


	node1.Start()
	client1.Start()
	client2.Start()

	time.Sleep(time.Millisecond * 300)
	if (client1.remoteNodes.Size() != 1) || (client2.remoteNodes.Size() != 0) {
		t.Error("node with mode PASSIVE should not be able to connect to each other")
	}

	node1.ShutDown()
	client1.ShutDown()
	client2.ShutDown()
}
