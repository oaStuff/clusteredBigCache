package cluster

import (
	"github.com/oaStuff/clusteredBigCache/comms"
	"time"
	"com.aihe/TOR/node"
	"github.com/oaStuff/clusteredBigCache/utils"
)

const (
	peerStateConnecting 	= 	iota
	peerStateConnected
	peerStateDisconnected
)

type remoteNodeState uint8

type remoteNodeConfig struct {
	Id        string `json:"id"`
	IpAddress string `json:"ip_address"`
}

type remoteNode struct {
	config        *remoteNodeConfig
	connection    *comms.Connection
	parentNode    *Node
	indexInParent int
	logger        utils.AppLogger
	state         remoteNodeState
}

func newRemoteNode(config *remoteNodeConfig, parent *Node, logger utils.AppLogger) *remoteNode {
	return &remoteNode{
		config:		config,
		state:		peerStateDisconnected,
		parentNode: parent,
		logger:		logger,
	}
}

func (r *remoteNode) join()  {
	utils.Info(r.logger, "Joining cluster via " + r.config.IpAddress )
	if err := r.connect(); err != nil {
		utils.Error(r.logger, err.Error())
	}
}

func (r *remoteNode) connect() error {
	var err error
	r.state = peerStateConnecting
	r.connection, err = comms.NewConnection(r.config.IpAddress, time.Second * 5)
	if err != nil {
		return err
	}

	r.parentNode.eventRemoteNodeConneced(r)
	r.state = peerStateConnected
	return nil
}

