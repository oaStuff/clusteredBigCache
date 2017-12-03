package cluster

import (
	"github.com/oaStuff/clusteredBigCache/utils"
	"github.com/oaStuff/clusteredBigCache/bigcache"
	"sync"
)

type NodeConfig struct {
	Id             string   `json:"id"`
	Join           bool     `json:"join"`
	JoinIp         string   `json:"join_ip"`
	LocalAddresses []string `json:"local_addresses"`
	LocalPort      uint     `json:"local_port"`
	BindAll        bool     `json:"bind_all"`
}

type Node struct {
	config      	*NodeConfig
	cache       	*bigcache.BigCache
	remoteNodes 	*utils.SliceList
	logger      	utils.AppLogger
	lock 			sync.Mutex
}

func NewNode(config *NodeConfig, logger utils.AppLogger) *Node {

	cache, err := bigcache.NewBigCache(bigcache.DefaultConfig())
	if err != nil {
		panic(err)
	}

	return &Node{
		config:			config,
		cache: 			cache,
		remoteNodes: 	utils.NewSliceList(),
		logger: 		logger,
		lock: 			sync.Mutex{},
	}
}

func (node *Node) Start() error  {

	if "" == node.config.Id {
		node.config.Id = GenerateNodeId(32)
		utils.Info(node.logger,"Node ID is " + node.config.Id)
	}

	node.bringNodeUp()
	if true == node.config.Join {	//we are to join an existing cluster
		node.joinCluster()
	}

	return nil
}

func (node *Node) joinCluster() {
	if "" == node.config.JoinIp {
		utils.Critical(node.logger,"The server' IP to join can not be empty.")
		panic("The server's IP to join can not be empty. Since Join is true, there must be a JoinIP")
	}

	remoteNode := newRemoteNode(&remoteNodeConfig{IpAddress:node.config.JoinIp}, node, node.logger)
	remoteNode.join()
}

func (node *Node) bringNodeUp() {
	utils.Info(node.logger, "Bringing up node " + node.config.Id)
}

func (node *Node) eventRemoteNodeConneced(remoteNode *remoteNode)  {
	node.lock.Lock()
	defer node.lock.Unlock()

	remoteNode.indexInParent = node.remoteNodes.Add(remoteNode)
}

func (node *Node) eventRemoteNodeDisconneced(remoteNode *remoteNode)  {
	node.lock.Lock()
	defer node.lock.Unlock()

	node.remoteNodes.Remove(remoteNode.indexInParent)
}
