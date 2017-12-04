package cluster

import (
	"github.com/oaStuff/clusteredBigCache/utils"
	"github.com/oaStuff/clusteredBigCache/bigcache"
	"sync"
	"net"
	"strconv"
	"github.com/oaStuff/clusteredBigCache/comms"
	"errors"
)

type NodeConfig struct {
	Id             string   `json:"id"`
	Join           bool     `json:"join"`
	JoinIp         string   `json:"join_ip"`
	LocalAddresses []string `json:"local_addresses"`
	LocalPort      int      `json:"local_port"`
	BindAll        bool     `json:"bind_all"`
}

type Node struct {
	config      	*NodeConfig
	cache       	*bigcache.BigCache
	remoteNodes 	*utils.SliceList
	logger      	utils.AppLogger
	lock 			sync.Mutex
	serverEndpoint 	net.Listener
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
		if err := node.joinCluster(); err != nil {
			return err
		}
	}

	return nil
}

func (node *Node) joinCluster() error {
	if "" == node.config.JoinIp {
		utils.Critical(node.logger,"the server' IP to join can not be empty.")
		return errors.New("the server's IP to join can not be empty since Join is true, there must be a JoinIP")
	}

	remoteNode := newRemoteNode(&remoteNodeConfig{IpAddress:node.config.JoinIp}, node, node.logger)
	remoteNode.join()

	return nil
}

func (node *Node) bringNodeUp() {
	utils.Info(node.logger, "bringing up node " + node.config.Id)
	go node.listen()
}

func (node *Node) eventRemoteNodeConneced(remoteNode *remoteNode)  {
	node.lock.Lock()
	defer node.lock.Unlock()

	remoteNode.indexInParent = node.remoteNodes.Add(remoteNode)
}

func (node *Node) eventRemoteNodeDisconneced(remoteNode *remoteNode)  {

	if remoteNode.indexInParent < 0 {
		return
	}

	node.lock.Lock()
	defer node.lock.Unlock()

	node.remoteNodes.Remove(remoteNode.indexInParent)
}

func (node *Node) listen() {

	var err error
	node.serverEndpoint, err = net.Listen("tcp",":" + strconv.Itoa(node.config.LocalPort))
	if err != nil {
		panic("unable to Listen on provided port number. [" + err.Error() + "]")
	}

	errCount := 0
	for {
		conn, err :=node.serverEndpoint.Accept()
		if err != nil {
			utils.Error(node.logger, err.Error())
			errCount++
			if errCount >= 5 {
				break
			}
			continue
		}
		//TODO: query the client for its details and insert into remoteNodes structure
		tcpConn := conn.(*net.TCPConn)
		remoteNode := newRemoteNode(&remoteNodeConfig{IpAddress:tcpConn.RemoteAddr().String()}, node, node.logger)
		remoteNode.setState(nodeStateHandshake)
		remoteNode.setConnection(comms.WrapConnection(tcpConn))
	}
	utils.Critical(node.logger, "listening loop terminated unexpectedly due to too many errors")
	panic("listening loop terminated unexpectedly due to too many errors")
}
