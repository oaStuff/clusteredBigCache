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

const (
	REPLICATION_MODE_FULL_REPLICATE byte = iota
	REPLICATION_MODE_SHARD

	clusterStateStarting byte = iota
	clusterStateStarted
	clusterStateEnded

	clusterModeACTIVE byte = iota
	clusterModePASSIVE
)

const CHAN_SIZE  = 1024 * 64

var (
	ErrNotEnoughReplica		=	errors.New("not enough replica")
	ErrNotFound				= 	errors.New("data not found")
	ErrTimedOut				= 	errors.New("not found as a result of timing out")
	ErrNotStarted			=	errors.New("node not started, call Start()")
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
	ReplicationMode	  byte	`json:"replication_mode"`
	ReplicationFactor int `json:"replication_factor"`
	WriteAck          bool   `json:"write_ack"`
	DebugMode		  bool	`json:"debug_mode"`
	DebugPort 		  int 	`json:"debug_port"`
	ReconnectOnDisconnect	bool	`json:"reconnect_on_disconnect"`
	PingFailureThreshHold int32  `json:"ping_failure_thresh_hold"`
	PingInterval          int    `json:"ping_interval"`
	PingTimeout           int    `json:"ping_timeout"`
}

//Cluster definition
type ClusteredBigCache struct {
	config         *ClusteredBigCacheConfig
	cache          *bigcache.BigCache
	remoteNodes    *utils.SliceList
	logger         utils.AppLogger
	serverEndpoint net.Listener
	joinQueue      chan *message.ProposedPeer
	pendingConn    sync.Map
	nodeIndex	   int
	getRequestChan	chan *getRequestDataWrapper
	replicationChan	chan *replicationMsg
	state 			byte
	mode 			byte
}

//create a new local node
func New(config *ClusteredBigCacheConfig, logger utils.AppLogger) *ClusteredBigCache {

	cfg := bigcache.DefaultConfig()
	cache, err := bigcache.NewBigCache(cfg)
	if err != nil {
		panic(err)
	}

	return &ClusteredBigCache{
		config:      config,
		cache:       cache,
		remoteNodes: utils.NewSliceList(),
		logger:      logger,
		joinQueue:   make(chan *message.ProposedPeer, 512),
		pendingConn: sync.Map{},
		nodeIndex: 	 0,
		getRequestChan:	 make(chan *getRequestDataWrapper, CHAN_SIZE),
		replicationChan: make(chan *replicationMsg, CHAN_SIZE),
		state: 			clusterStateStarting,
		mode: 			clusterModeACTIVE,
	}
}

//create a new local node that does not store any data locally
func NewPassiveClient(id string, serverEndpoint string, localPort, pingInterval, pingTimeout int, pingFailureThreashold int32, logger utils.AppLogger) *ClusteredBigCache {

	config := DefaultClusterConfig()
	config.Id = id
	config.Join = true
	config.JoinIp = serverEndpoint
	config.ReconnectOnDisconnect = true
	config.LocalPort = localPort
	config.PingInterval = pingInterval
	config.PingTimeout = pingTimeout
	config.PingFailureThreshHold = pingFailureThreashold

	return &ClusteredBigCache{
		config:      config,
		cache:       nil,
		remoteNodes: utils.NewSliceList(),
		logger:      logger,
		joinQueue:   make(chan *message.ProposedPeer, 512),
		pendingConn: sync.Map{},
		nodeIndex: 	 0,
		getRequestChan:	 make(chan *getRequestDataWrapper, CHAN_SIZE),
		replicationChan: make(chan *replicationMsg, CHAN_SIZE),
		state: 			clusterStateStarting,
		mode: 			clusterModePASSIVE,
	}
}


//check configuration values
func (node *ClusteredBigCache) checkConfig()  {
	if node.config.LocalPort < 1 {
		panic("Local port can not be zero.")
	}

	if node.config.ConnectRetries < 1 {
		node.config.ConnectRetries = 5
	}

	if node.config.ReplicationMode == REPLICATION_MODE_SHARD {
		utils.Warn(node.logger, "replication mode SHARD not yet implemented, falling back to FULL REPLICATION")
		node.config.ReplicationMode = REPLICATION_MODE_FULL_REPLICATE
	}

	if node.config.ReplicationMode == REPLICATION_MODE_SHARD && node.config.ReplicationFactor < 1 {
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

	for x := 0; x < 5; x++ {
		go node.requestSenderForGET()
		go node.replication()
	}

	node.checkConfig()
	if "" == node.config.Id {
		node.config.Id = utils.GenerateNodeId(32)
	}
	utils.Info(node.logger, "cluster node ID is " + node.config.Id)


	if err := node.bringNodeUp(); err != nil {
		return err
	}

	go node.connectToExistingNodes()
	if true == node.config.Join { //we are to join an existing cluster
		if err := node.joinCluster(); err != nil {
			return err
		}
	}

	node.state = clusterStateStarted
	time.Sleep(time.Millisecond * 200) //allow things to start up
	return nil
}

//shut down this Cluster and all terminate all connections to remoteNodes
func (node *ClusteredBigCache) ShutDown() {

	node.state = clusterStateEnded
	for _, v := range node.remoteNodes.Values() {
		rn := v.(*remoteNode)
		rn.config.ReconnectOnDisconnect = false
		rn.shutDown()
	}

	close(node.joinQueue)
	close(node.getRequestChan)
	close(node.replicationChan)

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
												Sync: true, ReconnectOnDisconnect: node.config.ReconnectOnDisconnect,
												PingInterval: node.config.PingInterval,
												PingTimeout: node.config.PingTimeout,
												PingFailureThreshHold: node.config.PingFailureThreshHold},
												node, node.logger)
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
		utils.Warn(node.logger, "clusterBigCache already contains " + remoteNode.config.Id)
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
														Sync: false, ReconnectOnDisconnect: false,
														PingInterval: node.config.PingInterval,
														PingTimeout: node.config.PingTimeout,
														PingFailureThreshHold: node.config.PingFailureThreshHold},
														node, node.logger)
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

//this is a goroutine that takes details from a channel and connect to them if they are not known
//when a remote system connects to this node or when this node connects to a remote system, it will query that system
//for the list of its connected nodes and pushes that list into this channel so that this node can connect forming
//a mesh network in the process
func (node *ClusteredBigCache) connectToExistingNodes() {

	for value := range node.joinQueue {

		if node.state != clusterStateStarted {
			continue
		}

		if _, ok := node.pendingConn.Load(value.Id); ok {
			utils.Warn(node.logger, fmt.Sprintf("remote node '%s' already in connnection pending queue", value.Id))
			continue
		}

		//if we are already connected to the remote node just continue
		keys := node.remoteNodes.Keys()
		if _, ok := keys.Load(value.Id); ok {
			continue
		}

		//we are here because we don't know this remote node
		remoteNode := newRemoteNode(&remoteNodeConfig{IpAddress: value.IpAddress,
			ConnectRetries: node.config.ConnectRetries,
			Id: value.Id, Sync: false, ReconnectOnDisconnect: node.config.ReconnectOnDisconnect}, node, node.logger)
		remoteNode.join()
		node.pendingConn.Store(value.Id, value.IpAddress)
	}
}

func (node *ClusteredBigCache) doShardReplication(key string, data []byte, duration time.Duration) error {
	panic("shard replication not yet implemented")

	if node.config.ReplicationFactor == 1 {
		_, err := node.cache.Set(key, data, duration)
		return err
	}

	if node.remoteNodes.Size() < int32(node.config.ReplicationFactor -1) {
		return ErrNotEnoughReplica
	}
	return nil
}

//puts the data into the cluster
func (node *ClusteredBigCache) Put(key string, data []byte, duration time.Duration) error {

	if node.state != clusterStateStarted {
		return ErrNotStarted
	}

	if node.config.ReplicationMode == REPLICATION_MODE_SHARD {
		return node.doShardReplication(key, data, duration)
	}

	//store it locally first
	expiryTime := bigcache.NO_EXPIRY
	if node.mode == clusterModeACTIVE {
		var err error
		expiryTime, err = node.cache.Set(key, data, duration)
		if err != nil {
			return err
		}
	} else if node.mode == clusterModePASSIVE {
		expiryTime = uint64(time.Now().Unix())
		if duration != time.Duration(bigcache.NO_EXPIRY) {
			expiryTime += uint64(duration.Seconds())
		} else {
			expiryTime = bigcache.NO_EXPIRY
		}
	}

	//we are going to do full replication across the cluster
	peers := node.remoteNodes.Values()
	for x := 0; x < len(peers); x++ {	//just replicate serially from left to right
		if peers[x].(*remoteNode).mode == clusterModePASSIVE {
			continue
		}
		node.replicationChan <- &replicationMsg{r: peers[x].(*remoteNode),
			m: &message.PutMessage{Key: key, Data: data, Expiry: expiryTime}}
	}

	return nil
}

//gets the data from the cluster
func (node *ClusteredBigCache) Get(key string, timeout time.Duration) ([]byte, error) {
	if node.state != clusterStateStarted {
		return nil, ErrNotStarted
	}

	//if present locally then send it
	if node.mode == clusterModeACTIVE {
		data, err := node.cache.Get(key)
		if err == nil {
			return data, nil
		}
	}

	//we did not get the data locally so lets check the cluster
	peers := node.getRemoteNodes()
	if len(peers) < 1 {
		return nil, ErrNotFound
	}
	replyC := make(chan *getReplyData)
	reqData := &getRequestData{key: key, randStr: utils.GenerateNodeId(8),
									replyChan: replyC, done: make(chan struct{})}


	for _, peer := range peers {
		if peer.(*remoteNode).mode == clusterModePASSIVE {
			continue
		}
		node.getRequestChan <- &getRequestDataWrapper{r: peer.(*remoteNode), g: reqData}
	}

	var replyData *getReplyData
	select {
	case replyData = <-replyC:
	case <-time.After(timeout):
		return nil, ErrTimedOut
	}

	close(reqData.done)
	return replyData.data, nil
}

//delete a key from the cluster
func (node *ClusteredBigCache) Delete(key string) error {

	if node.state != clusterStateStarted {
		return ErrNotStarted
	}

	//delete locally
	if node.mode == clusterModeACTIVE {
		node.cache.Delete(key)
	}

	peers := node.remoteNodes.Values()
	//just send the delete message to everyone
	for x := 0; x < len(peers); x++ {
		if peers[x].(*remoteNode).mode == clusterModePASSIVE {
			continue
		}
		node.replicationChan <- &replicationMsg{r:peers[x].(*remoteNode), m: &message.DeleteMessage{Key: key}}
	}



	return nil
}

func (node *ClusteredBigCache) Statistics() string {
	if node.mode == clusterModeACTIVE {
		stats := node.cache.Stats()
		return fmt.Sprintf("%q", stats)
	}

	return "No stats for passive mode"
}

//a goroutine to send get request to members of the cluster
func (node *ClusteredBigCache) requestSenderForGET() {
	for value := range node.getRequestChan {
		value.r.getData(value.g)
	}
}

//a goroutine used to replicate messages across the cluster
func (node *ClusteredBigCache) replication() {
	for msg := range node.replicationChan {
		msg.r.sendMessage(msg.m)
	}
}

