package clusteredBigCache

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"encoding/binary"
	"github.com/oaStuff/clusteredBigCache/comms"
	"github.com/oaStuff/clusteredBigCache/message"
	"github.com/oaStuff/clusteredBigCache/utils"
)

const (
	nodeStateConnecting = iota
	nodeStateConnected
	nodeStateDisconnected
	nodeStateShuttingDown
	nodeStateHandshake
)

type remoteNodeState uint8

type nodeMetrics struct {
	pingSent		uint64
	pingRecieved	uint64
	pongSent		uint64
	pongRecieved	uint64
	dropedMsg		uint64
}

// remote node configuration
type remoteNodeConfig struct {
	Id                    string `json:"id"`
	IpAddress             string `json:"ip_address"`
	PingFailureThreshHold int32  `json:"ping_failure_thresh_hold"`
	PingInterval          int    `json:"ping_interval"`
	PingTimeout           int    `json:"ping_timeout"`
	ConnectRetries        int    `json:"connect_retries"`
	ServicePort           string `json:"service_port"`
	Sync                  bool   `json:"sync"`
}

// remote node definition
type remoteNode struct {
	config        *remoteNodeConfig
	metrics 	  *nodeMetrics
	connection    *comms.Connection
	parentNode    *ClusteredBigCache
	msgQueue      chan *message.NodeWireMessage
	indexInParent int
	logger        utils.AppLogger
	state         remoteNodeState
	stateLock     sync.Mutex
	done          chan struct{}
	pingTimer     *time.Ticker //used to send ping message to remote
	pingTimeout   *time.Timer  //used to monitor ping response
	pingFailure   int32        //count the number of pings without response
}

//used by SliceList to check for equality of items in the list
func remoteNodeEqualFunc(item1, item2 interface{}) bool {
	if (nil == item1) || (nil == item2) {
		return false
	}
	return item1.(*remoteNode).config.Id == item2.(*remoteNode).config.Id
}

//used by SliceList to get the key of objects stored in the list
func remoteNodeKeyFunc(item interface{}) string {
	if nil == item {
		return ""
	}

	return item.(*remoteNode).config.Id
}

//check configurations for sensible defaults
func checkConfig(logger utils.AppLogger, config *remoteNodeConfig) {

	if config.PingInterval < 1 {
		config.PingInterval = 5
	}

	if config.PingTimeout < 1 {
		config.PingTimeout = 3
	}

	if config.PingTimeout > config.PingInterval {
		utils.Warn(logger, "ping timeout is greater than ping interval, pings will NEVER timeout")
	}

	if config.PingFailureThreshHold == 0 {
		config.PingFailureThreshHold = 5
	}
}

//create a new remoteNode object
func newRemoteNode(config *remoteNodeConfig, parent *ClusteredBigCache, logger utils.AppLogger) *remoteNode {
	checkConfig(logger, config)
	return &remoteNode{
		config:        config,
		msgQueue:      make(chan *message.NodeWireMessage, 1024),
		done:          make(chan struct{}, 2),
		state:         nodeStateDisconnected,
		stateLock:     sync.Mutex{},
		parentNode:    parent,
		logger:        logger,
		indexInParent: -1,
		metrics: 		&nodeMetrics{},
	}
}

//set the state of this remoteNode. always use this method because of the lock
func (r *remoteNode) setState(state remoteNodeState) {
	r.stateLock.Lock()
	defer r.stateLock.Unlock()

	r.state = state
}

//just set the connection for this remoteNode
func (r *remoteNode) setConnection(conn *comms.Connection) {
	r.connection = conn
}

//set up the pinging and ping response go routine
func (r *remoteNode) startPinging() {
	r.pingTimer = time.NewTicker(time.Second * time.Duration(r.config.PingInterval))
	r.pingTimeout = time.NewTimer(time.Second * time.Duration(r.config.PingTimeout))

	r.sendMessage(&message.PingMessage{}) //send the first ping message

	go func() {
		done := false
		for {
			select {
			case <-r.pingTimer.C:
				r.sendMessage(&message.PingMessage{})
				atomic.AddUint64(&r.metrics.pingSent, 1)
				r.pingTimeout.Stop()
				r.pingTimeout.Reset(time.Second * time.Duration(r.config.PingTimeout))
			case <-r.done: //we have this so that the goroutine would not linger after this node disconnects because
				done = true //of the blocking channel in the above case statement
			}

			if done {
				break
			}
		}

		utils.Info(r.logger, fmt.Sprintf("shutting down ping timer goroutine for '%s'", r.config.Id))
	}()

	go func() {
		done := false
		fault := false
		for {
			select {
			case <-r.pingTimeout.C:
				if r.state != nodeStateHandshake {
					utils.Warn(r.logger,
						fmt.Sprintf("no ping response within configured time frame from remote node '%s'", r.config.Id))
				} else {
					utils.Warn(r.logger, "remote node not verified, therefore ping failing")
				}
				atomic.AddInt32(&r.pingFailure, 1)
				if r.pingFailure >= r.config.PingFailureThreshHold {
					r.pingTimeout.Stop()
					fault = true
					break
				}
			case <-r.done: //we have this so that the goroutine would not linger after this node disconnects because
				done = true //of the blocking channel in the above case statement
			}

			if done || fault {
				break
			}
		}

		if fault {
			//the remote node is assumed to be 'dead' since it has not responded to recent ping request
			utils.Warn(r.logger, fmt.Sprintf("shutting down connection to remote node '%s' due to no ping response", r.config.Id))
			r.shutDown()
		}
		utils.Info(r.logger, fmt.Sprintf("shutting down ping timeout goroutine for '%s'", r.config.Id))
	}()
}

//kick start this remoteNode entity
func (r *remoteNode) start() {
	go r.networkConsumer()
	go r.handleMessage()
	r.sendVerify()
	r.startPinging() //start this early here so that clients that connected without responding to PINGS will be diconnected
	//ping response from clients has not yet sent MsgVERIFY will be discarded
}

//join a cluster. this will be called if 'join' in the config is set to true
func (r *remoteNode) join() {
	utils.Info(r.logger, "joining remote node via "+r.config.IpAddress)

	go func() { //goroutine will try to connect to the cluster until it succeeds or max tries reached
		var err error
		tries := 0
		for {
			if err = r.connect(); err == nil {
				break
			}
			utils.Error(r.logger, err.Error())
			time.Sleep(time.Second * 3)
			if r.config.ConnectRetries > 0 {
				tries++
				if tries >= r.config.ConnectRetries {
					utils.Warn(r.logger, fmt.Sprintf("unable to connect to remote node '%s' after max retires", r.config.IpAddress))
					r.parentNode.eventUnableToConnect(r.config)
					return
				}
			}
		}
		utils.Info(r.logger, "connected to node via "+r.config.IpAddress)
		r.start()
	}()
}

//handles to low level connection to remote node
func (r *remoteNode) connect() error {
	var err error
	utils.Info(r.logger, "connecting to "+r.config.IpAddress)
	r.connection, err = comms.NewConnection(r.config.IpAddress, time.Second*5)
	if err != nil {
		return err
	}

	r.setState(nodeStateHandshake)

	return nil
}

//this is a goroutine dedicated in reading data fromt the network
//it does this by reading a 4bytes header which is
//	byte 1 & 2 == length of data
//	byte 3 & 4 == message code
//	the rest of the data based on length is the message body
func (r *remoteNode) networkConsumer() {

	for (r.state == nodeStateConnected) || (r.state == nodeStateHandshake) {

		header, err := r.connection.ReadData(4, 0) //read 4 byte header
		if nil != err {
			utils.Critical(r.logger, fmt.Sprintf("remote node '%s' has disconnected", r.config.Id))
			r.shutDown()
			return
		}

		dataLength := int16(binary.LittleEndian.Uint16(header)) - 2 //subtracted 2 becos of message code
		msgCode := binary.LittleEndian.Uint16(header[2:])
		var data []byte = nil
		if dataLength > 0 {
			data, err = r.connection.ReadData(uint(dataLength), 0)
			if nil != err {
				utils.Critical(r.logger, fmt.Sprintf("remote node '%s' has disconnected", r.config.Id))
				r.shutDown()
				return
			}
		}
		r.queueMessage(&message.NodeWireMessage{Code: msgCode, Data: data}) //queue message to be processed
	}

}

//this sends a message on the network
//it builds the message using the following protocol
// bytes 1 & 2 == total length of the data (including the 2 byte message code)
// bytes 3 & 4 == message code
// bytes 5 upwards == message content
func (r *remoteNode) sendMessage(m message.NodeMessage) {
	msg := m.Serialize()
	data := make([]byte, 4+len(msg.Data))
	binary.LittleEndian.PutUint16(data, uint16(len(msg.Data)+2)) //the 2 is for the message code
	binary.LittleEndian.PutUint16(data[2:], msg.Code)
	copy(data[4:], msg.Data)
	if err := r.connection.SendData(data); err != nil {
		utils.Critical(r.logger, "unexpected error while sending data ["+err.Error()+"]")
		r.shutDown()
	}
}

//bring down the remote node. should not be called from outside networkConsumer()
func (r *remoteNode) shutDown() {
	r.stateLock.Lock()
	defer r.stateLock.Unlock()

	if r.state == nodeStateDisconnected {
		return
	}

	r.state = nodeStateDisconnected
	r.parentNode.eventRemoteNodeDisconneced(r)
	r.connection.Close()
	r.parentNode = nil
	if r.pingTimeout != nil {
		r.pingTimeout.Stop()
	}
	if r.pingTimer != nil {
		r.pingTimer.Stop()
	}
	close(r.done)
	close(r.msgQueue)
	utils.Info(r.logger, fmt.Sprintf("shutting down remote node '%s' ", r.config.Id))
}

//just queue the message in a channel
func (r *remoteNode) queueMessage(msg *message.NodeWireMessage) {

	if r.state == nodeStateHandshake { //when in the handshake state only accept MsgVERIFY and MsgVERIFYOK messages
		code := msg.Code
		if (code != message.MsgVERIFY) && (code != message.MsgVERIFYOK) {
			r.metrics.dropedMsg++
			return
		}
	}

	if r.state != nodeStateDisconnected {
		r.msgQueue <- msg
		//we could modify handleMessage() to run async as in go handleMessage(msg)
		//instead of queuing the message
	}
}

//message handler
func (r *remoteNode) handleMessage() {

	for msg := range r.msgQueue {
		switch msg.Code {
		case message.MsgVERIFY:
			r.handleVerify(msg)
		case message.MsgVERIFYOK:
			r.handleVerifyOK()
		case message.MsgPING:
			r.handlePing()
		case message.MsgPONG:
			r.handlePong()
		case message.MsgSyncRsp:
			r.handleSyncResponse(msg)
		case message.MsgSyncReq:
			r.handleSyncRequest()

		}
	}

	utils.Info(r.logger, fmt.Sprintf("terminated message handler goroutine for '%s'", r.config.Id))
}

//send a verify messge. this is always the first message to be sent once a connection is established.
func (r *remoteNode) sendVerify() {
	verifyMsgRsp := message.VerifyMessage{Id: r.parentNode.config.Id,
		ServicePort: strconv.Itoa(r.parentNode.config.LocalPort)}
	r.sendMessage(&verifyMsgRsp)
}

//use the verify message been sent by a remote node to configure the node in this system
func (r *remoteNode) handleVerify(msg *message.NodeWireMessage) {

	verifyMsgRsp := message.VerifyMessage{}
	verifyMsgRsp.DeSerialize(msg)
	r.config.Id = verifyMsgRsp.Id
	r.config.ServicePort = verifyMsgRsp.ServicePort
	if !r.parentNode.eventVerifyRemoteNode(r) { //seek parent's node approval on this
		utils.Warn(r.logger, fmt.Sprintf("node already has remote node '%s' so shutdown new connection", r.config.Id))
		r.shutDown()
		return
	}

	r.setState(nodeStateConnected)
	r.sendMessage(&message.VerifyOKMessage{}) //must reply back with a verify OK message if all goes well
}

//handles verify OK from a remote node. this allows this system to sync with remote node
func (r *remoteNode) handleVerifyOK() {
	go func() {
		count := 0
		for r.state == nodeStateHandshake {
			time.Sleep(time.Second * 1)
			count++
			if count >= 5 {
				utils.Warn(r.logger, fmt.Sprintf("node '%s' state refused to change out of handshake", r.config.Id))
				break
			}
		}
		if count < 5 {
			if r.config.Sync { //only sync if you are joining the cluster
				r.sendMessage(&message.SyncReqMessage{})
			}
		}
	}()

}

//handles ping message from a remote node
func (r *remoteNode) handlePing() {
	atomic.AddUint64(&r.metrics.pingRecieved, 1)
	r.sendMessage(&message.PongMessage{})
	atomic.AddUint64(&r.metrics.pongSent, 1)
}

//handle a pong message from the remote node, reset flags
func (r *remoteNode) handlePong() {
	atomic.AddUint64(&r.metrics.pongRecieved, 1)
	r.pingTimeout.Stop()                 //stop the timer since we got a response
	atomic.StoreInt32(&r.pingFailure, 0) //reset failure counter since we got a response
}

//build and send a sync message
func (r *remoteNode) sendSyncResponse() {
	values := r.parentNode.getRemoteNodes() //call this because of the lock that needs to be held by parentNode
	length := len(values) - 1               //minus 1 beacause we dont want to send our own info
	nodeList := make([]message.ProposedPeer, length, length)
	x := 0
	for _, v := range values {
		n := v.(*remoteNode)
		if n.config.Id == r.config.Id {
			continue
		}
		host, _, _ := net.SplitHostPort(n.config.IpAddress)
		nodeList[x] = message.ProposedPeer{Id: n.config.Id, IpAddress: net.JoinHostPort(host, n.config.ServicePort)}
		x++
	}

	if len(nodeList) > 0 {
		r.sendMessage(&message.SyncRspMessage{List: nodeList, ReplicationFactor: r.parentNode.config.ReplicationFactor})
	}
}

//handles sync request by just sending a sync response
func (r *remoteNode) handleSyncRequest() {
	r.sendSyncResponse()
}

//accept the sync response and send to parentNode for processing
func (r *remoteNode) handleSyncResponse(msg *message.NodeWireMessage) {
	syncMsg := message.SyncRspMessage{}
	syncMsg.DeSerialize(msg)
	length := len(syncMsg.List)
	for x := 0; x < length; x++ {
		r.parentNode.joinQueue <- &syncMsg.List[x]
	}
}
