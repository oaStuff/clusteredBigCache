package clusteredBigCache

import (
	"github.com/oaStuff/clusteredBigCache/utils"
	"testing"
	"time"
)

func TestCheckConfig(t *testing.T) {
	config := &remoteNodeConfig{}
	checkConfig(nil, config)

	if config.PingInterval != 5 {
		t.Error("config ping interval ought to be 5")
	}

	if config.PingTimeout != 3 {
		t.Error("config ping timeout ought to be 3")
	}

	if config.PingFailureThreshHold != 5 {
		t.Error("config ping failure threshold ought to be 5")
	}
}

func TestRemoteNode(t *testing.T) {

	svr := utils.NewTestServer(9091, true)
	svr.Start()
	defer svr.Close()

	node := New(&ClusteredBigCacheConfig{LocalPort: 9999, ConnectRetries: 2}, nil)
	rn := newRemoteNode(&remoteNodeConfig{IpAddress: "localhost:9091", Sync: false}, node, nil)

	err := rn.connect()
	if err != nil {
		t.Error(err)
		return
	}

	if rn.state != nodeStateHandshake {
		t.Error("node ought to be in handshaking state")
		return
	}

	rn.start()
	defer rn.tearDown()

	svr.SendVerifyMessage("server1")
	time.Sleep(time.Millisecond * 300)
	if rn.state != nodeStateConnected {
		t.Error("node ought to be in connected state")
	}

	if node.remoteNodes.Size() != 1 {
		t.Error("only one node ought to be connected")
	}

	if node.remoteNodes.Values()[0].(*remoteNode).config.Id != "server1" {
		t.Error("unknown id in remoteNode data")
	}

	rn.shutDown()
	node.ShutDown()
}

func TestNoPingResponseDisconnt(t *testing.T) {
	svr := utils.NewTestServer(9092, false)
	err := svr.Start()
	if err != nil {
		panic(err)
	}
	defer svr.Close()

	node := New(&ClusteredBigCacheConfig{LocalPort: 9999, ConnectRetries: 2}, nil)
	rn := newRemoteNode(&remoteNodeConfig{IpAddress: "localhost:9092", Sync: false,
		PingFailureThreshHold: 1, PingInterval: 0}, node, nil)
	err = rn.connect()
	if err != nil {
		t.Error(err)
		return
	}

	if rn.state != nodeStateHandshake {
		t.Error("node ought to be in handshaking state")
	}

	rn.start()
	defer rn.tearDown()
	svr.SendVerifyMessage("server_1")

	time.Sleep(time.Second * 6)
	if rn.state != nodeStateDisconnected {
		t.Error("node ought to be in disconnected state")
	}

	if rn.metrics.pongRecieved != 0 {
		t.Error("pong ought not to have been recieved")
	}

	rn.shutDown()
	node.ShutDown()
}

func TestPinging(t *testing.T) {
	s := utils.NewTestServer(8999, true)
	err := s.Start()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	node := New(&ClusteredBigCacheConfig{LocalPort: 8999, ConnectRetries: 2, PingTimeout: 1, PingInterval: 2}, nil)
	time.Sleep(time.Millisecond * 200)
	rn := newRemoteNode(&remoteNodeConfig{IpAddress: "localhost:8999", Sync: true,
		PingFailureThreshHold: 10, PingInterval: 2, PingTimeout: 1}, node, nil)
	err = rn.connect()
	if err != nil {
		t.Error(err)
		return
	}

	if rn.state != nodeStateHandshake {
		t.Error("node ought to be in handshaking state")
	}

	rn.start()
	defer rn.tearDown()
	s.SendVerifyMessage("server1")

	time.Sleep(time.Second * 3)

	if rn.metrics.pongRecieved < 1 {
		t.Error("pong ought to have been recieved")
	}

	if rn.metrics.pingRecieved < 1 {
		t.Error("pinging facility not working approprately")
	}

	rn.shutDown()
	node.ShutDown()
}
