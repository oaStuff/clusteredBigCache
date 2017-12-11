package clusteredBigCache

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/oaStuff/clusteredBigCache/bigcache"
	"github.com/oaStuff/clusteredBigCache/comms"
	"github.com/oaStuff/clusteredBigCache/message"
	"github.com/oaStuff/clusteredBigCache/utils"
	"time"
)

var (
	ErrNotEnoughReplica		=	errors.New("not enough replica")
	ErrNotFound				= 	errors.New("data not found")
)

//Cluster configuration
type ClusteredBigCacheConfig struct {
	Id             string   `json:"id"`
	Join           bool     `json:"join"`
	JoinIp         string   `json:"join_ip"`
	LocalAddresses []string `json:"local_addresses"`
	LocalPort      int      `json:"local_port"`
	BindAll        bool     `json:"bind_all"`
	ConnectRetries int      `json:"connect_retries"`
	TerminateOnListenerExit	bool 	`json:"terminate_on_listener_exit"`
	ReplicationFactor int `json:"replication_factor"`
	WriteAck          bool   `json:"write_ack"`
	DebugMode		  bool	`json:"debug_mode"`
	DebugPort 		  int 	`json:"debug_port"`
}

//Cluster definition
type ClusteredBigCache struct {
	config         *ClusteredBigCacheConfig
	cache          *bigcache.BigCache
	remoteNodes    *utils.SliceList
	logger         utils.AppLogger
	//lock           sync.Mutex
	serverEndpoint net.Listener
	joinQueue      chan *message.ProposedPeer
	pendingConn    sync.Map
	nodeIndex	   int
	replicationLock sync.Mutex
	getRequestChan	chan *getRequestDataWrapper
}

//create a new local node
func New(config *ClusteredBigCacheConfig, logger utils.AppLogger) *ClusteredBigCache {

	cache, err := bigcache.NewBigCache(bigcache.DefaultConfig())
	if err != nil {
		panic(err)
	}

	return &ClusteredBigCache{
		config:      config,
		cache:       cache,
		remoteNodes: utils.NewSliceList(),
		logger:      logger,
		//lock:        sync.Mutex{},
		joinQueue:   make(chan *message.ProposedPeer, 512),
		pendingConn: sync.Map{},
		nodeIndex: 	 0,
		replicationLock: sync.Mutex{},
		getRequestChan:	 make(chan *getRequestDataWrapper, 1024),
	}
}

func (node *ClusteredBigCache) checkConfig()  {
	if node.config.LocalPort < 1 {
		panic("Local port can not be zero.")
	}

	if node.config.ReplicationFactor < 1 {
		utils.Warn(node.logger, "Adjusting replication to 1 (no replication) because it was less than 1")
		node.config.ReplicationFactor = 1
	}
}

func (node *ClusteredBigCache) setReplicationFactor(rf int)  {
	if rf < 1 {
		rf = 1
	}

	node.config.ReplicationFactor = rf
}

//start this Cluster running
func (node *ClusteredBigCache) Start() error {

	if node.config.DebugMode {
		go node.startUpHttpServer()
	}

	for x := 0; x < 10; x++ {
		go node.getRequestSender()
	}

	node.checkConfig()
	if "" == node.config.Id {
		node.config.Id = utils.GenerateNodeId(32)
		utils.Info(node.logger, "Cluster ID is "+node.config.Id)
	}

	if err := node.bringNodeUp(); err != nil {
		return err
	}

	go node.connectToExistingNodes()
	if true == node.config.Join { //we are to join an existing cluster
		if err := node.joinCluster(); err != nil {
			return err
		}
	}

	return nil
}

//shut down this Cluster and all terminate all connections to remoteNodes
func (node *ClusteredBigCache) ShutDown() {
	for _, v := range node.remoteNodes.Values() {
		v.(*remoteNode).shutDown()
	}

	close(node.joinQueue)
	close(node.getRequestChan)
	if node.serverEndpoint != nil {
		node.serverEndpoint.Close()
	}
}

//join an existing cluster
func (node *ClusteredBigCache) joinCluster() error {
	if "" == node.config.JoinIp {
		utils.Critical(node.logger, "the server's IP to join can not be empty.")
		return errors.New("the server's IP to join can not be empty since Join is true, there must be a JoinIP")
	}

	remoteNode := newRemoteNode(&remoteNodeConfig{IpAddress: node.config.JoinIp,
												ConnectRetries: node.config.ConnectRetries,
												Sync: true}, node, node.logger)
	remoteNode.join()
	return nil
}

//bring up this Cluster
func (node *ClusteredBigCache) bringNodeUp() error {

	var err error
	utils.Info(node.logger, "bringing up node " + node.config.Id)
	node.serverEndpoint, err = net.Listen("tcp", ":" + strconv.Itoa(node.config.LocalPort))
	if err != nil {
		utils.Error(node.logger, fmt.Sprintf("unable to Listen on port %d. [%s]", node.config.LocalPort, err.Error()))
		return err
	}

	go node.listen()
	return nil
}

//event function used by remoteNode to announce the disconnection of itself
func (node *ClusteredBigCache) eventRemoteNodeDisconneced(r *remoteNode) {

	node.remoteNodes.Remove(r.config.Id)
}

//util function to return all know remoteNodes
func (node *ClusteredBigCache) getRemoteNodes() []interface{} {

	return node.remoteNodes.Values()
}

//event function used by remoteNode to verify itself
func (node *ClusteredBigCache) eventVerifyRemoteNode(remoteNode *remoteNode) bool {

	if node.remoteNodes.Contains(remoteNode.config.Id) {
		return false
	}

	node.remoteNodes.Add(remoteNode.config.Id, remoteNode)
	utils.Info(node.logger, fmt.Sprintf("added remote node '%s' into group", remoteNode.config.Id))
	node.pendingConn.Delete(remoteNode.config.Id)

	return true
}

//event function used by remoteNode to notify this node of a connection that failed
func (node *ClusteredBigCache) eventUnableToConnect(config *remoteNodeConfig) {
	node.pendingConn.Delete(config.Id)
}

//listen for new connections to this node
func (node *ClusteredBigCache) listen() {

	utils.Info(node.logger, fmt.Sprintf("node '%s' is up and running", node.config.Id))
	errCount := 0
	for {
		conn, err := node.serverEndpoint.Accept()
		if err != nil {
			utils.Error(node.logger, err.Error())
			errCount++
			if errCount >= 5 {
				break
			}
			continue
		}
		errCount = 0

		//build a new remoteNode from this new connection
		tcpConn := conn.(*net.TCPConn)
		remoteNode := newRemoteNode(&remoteNodeConfig{IpAddress: tcpConn.RemoteAddr().String(),
														ConnectRetries: node.config.ConnectRetries,
														Sync: false}, node, node.logger)
		remoteNode.setState(nodeStateHandshake)
		remoteNode.setConnection(comms.WrapConnection(tcpConn))
		utils.Info(node.logger, fmt.Sprintf("new connection from remote '%s'", tcpConn.RemoteAddr().String()))
		remoteNode.start()
	}
	utils.Critical(node.logger, "listening loop terminated unexpectedly due to too many errors")
	if node.config.TerminateOnListenerExit {
		panic("listening loop terminated unexpectedly due to too many errors")
	}
}

func (node *ClusteredBigCache) DoTest() {
	fmt.Printf("list size is : %+v\n", node.remoteNodes.Size())
}

//this is a goroutine that takes details from a channel and connect to them if they are not known
//when a remote system connects to this node or when this node connects to a remote system, it will query that system
//for the list of its connected nodes and pushes that list into this channel so that this node can connect forming
//a mesh network in the process
func (node *ClusteredBigCache) connectToExistingNodes() {

	for value := range node.joinQueue {
		if _, ok := node.pendingConn.Load(value.Id); ok {
			utils.Warn(node.logger, fmt.Sprintf("remote node '%s' already in connnection pending queue", value.Id))
			continue
		}

		//if we are already connected to the remote node just exit
		keys := node.remoteNodes.Keys()
		if _, ok := keys.Load(value.Id); ok {
			continue
		}

		//we are here because we don't know this remote node
		remoteNode := newRemoteNode(&remoteNodeConfig{IpAddress: value.IpAddress,
			ConnectRetries: node.config.ConnectRetries,
			Id: value.Id, Sync: false}, node, node.logger)
		remoteNode.join()
		node.pendingConn.Store(value.Id, value.IpAddress)
	}
}

func (node *ClusteredBigCache) Put(key string, data []byte, duration time.Duration) error {
	if node.config.ReplicationFactor == 1 {
		_, err := node.cache.Set(key, data, duration)
		return err
	}

	if node.remoteNodes.Size() < int32(node.config.ReplicationFactor -1) {
		return ErrNotEnoughReplica
	}
	peers := node.remoteNodes.Values()
	expiryTime, err := node.cache.Set(key, data, duration)
	if err != nil {
		return err
	}

	node.replicationLock.Lock()
	defer node.replicationLock.Unlock()

	for x := 0; x < node.config.ReplicationFactor - 1; x++ {	//just replicate serially from left to right
		//fmt.Printf("replicating key '%s' to node '%s'\n", key, peers[node.nodeIndex].(*remoteNode).config.Id)
		peers[node.nodeIndex].(*remoteNode).sendMessage(&message.PutMessage{Key: key, Data: data, Expiry: expiryTime})
		node.nodeIndex = (node.nodeIndex + 1) % len(peers)
	}

	return nil
}

func (node *ClusteredBigCache) Get(key string, timeout time.Duration) ([]byte, error) {
	data, err := node.cache.Get(key)
	if err == nil {
		return data, nil
	}

	//we did not get the data locally so lets check the cluster
	peers := node.getRemoteNodes()
	replyC := make(chan *getReplyData)
	reqData := &getRequestData{key: key, randStr: utils.GenerateNodeId(8),
									replyChan: replyC, done: make(chan struct{})}

	//TODO: should we send to every remote node?
	for _, peer := range peers {
		node.getRequestChan <- &getRequestDataWrapper{r: peer.(*remoteNode), g: reqData}
	}

	var replyData *getReplyData
	select {
	case replyData = <-replyC:
	case <-time.After(timeout):
		return nil, ErrNotFound
	}
	//close(reqData.replyChan)
	close(reqData.done)
	return replyData.data, nil
}

func (node *ClusteredBigCache) Delete(key string) error {
	return nil
}

func (node *ClusteredBigCache) getRequestSender() {
	for value := range node.getRequestChan {
		value.r.getData(value.g)
	}
}
