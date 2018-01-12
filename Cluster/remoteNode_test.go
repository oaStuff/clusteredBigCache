package clusteredBigCache

import (
	"testing"
	"time"
	"github.com/oaStuff/clusteredBigCache/utils"
)

func TestCheckConfig(t *testing.T)  {
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

func TestRemoteNode(t *testing.T)  {

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
	time.Sleep(time.Second * 1)
	if rn.state != nodeStateConnected {
		t.Error("node ought to be in connected state")
	}

	if node.remoteNodes.Size() != 1 {
		t.Error("only one node ought to be connected")
	}

	if node.remoteNodes.Values()[0].(*remoteNode).config.Id != "server1" {
		t.Error("unknown id in remoteNode data")
	}
}

func TestNoPingResponseDisconnt(t *testing.T)  {
	s := utils.NewTestServer(9092, false)
	err := s.Start()
	if err != nil {
		panic(err)
	}
	defer s.Close()


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

	time.Sleep(time.Second * 6)
	if rn.state != nodeStateDisconnected {
		t.Error("node ought to be in disconnected state")
	}

	if rn.metrics.pongRecieved != 0 {
		t.Error("pong ought not to have been recieved")
	}

	if rn.metrics.dropedMsg == 0 {
		t.Error("pinging facility not working approprately")
	}
}
