package cluster

import (
	"github.com/oaStuff/clusteredBigCache/comms"
	"time"
	"github.com/oaStuff/clusteredBigCache/utils"
	"encoding/binary"
	"github.com/oaStuff/clusteredBigCache/message"
	"fmt"
	"sync/atomic"
	"sync"
	"strconv"
	"net"
)

const (
	nodeStateConnecting         = 	iota
	nodeStateConnected
	nodeStateDisconnected
	nodeStateShuttingDown
	nodeStateHandshake
)

type remoteNodeState uint8


// remote node configuration
type remoteNodeConfig struct {
	Id                    	string 		`json:"id"`
	IpAddress             	string 		`json:"ip_address"`
	PingFailureThreshHold 	int32  		`json:"ping_failure_thresh_hold"`
	PingInterval          	int    		`json:"ping_interval"`
	PingTimeout           	int    		`json:"ping_timeout"`
	ConnectRetries        	int    		`json:"connect_retries"`
	ServicePort				string 		`json:"service_port"`
}


// remote node definition
type remoteNode struct {
	config        		*remoteNodeConfig
	connection    		*comms.Connection
	parentNode    		*Node
	msgQueue 	  		chan *message.NodeWireMessage
	indexInParent 		int
	logger        		utils.AppLogger
	state         		remoteNodeState
	stateLock			sync.Mutex
	done				chan struct{}
	pingTimer			*time.Ticker		//used to send ping message to remote
	pingTimeout			*time.Timer			//used to monitor ping response
	pingFailure			int32				//count the number of pings without response
}

func remoteNodeEqualFunc(item1, item2 interface{}) bool {
	if (nil == item1) || (nil == item2){
		return false
	}
	return item1.(*remoteNode).config.Id == item2.(*remoteNode).config.Id
}

func remoteNodeKeyFunc(item interface{}) string {
	if nil == item {
		return ""
	}

	return item.(*remoteNode).config.Id
}

func checkConfig(logger utils.AppLogger, config *remoteNodeConfig)  {

	if config.PingInterval < 5 {
		config.PingInterval = 5
	}

	if config.PingTimeout < 3 {
		config.PingTimeout = 3
	}

	if config.PingTimeout > config.PingInterval {
		utils.Warn(logger, "ping timeout is greater than ping interval, pings will NEVER timeout")
	}

	if config.PingFailureThreshHold < 5 {
		config.PingFailureThreshHold = 5
	}
}

func newRemoteNode(config *remoteNodeConfig, parent *Node, logger utils.AppLogger) *remoteNode {
	checkConfig(logger, config)
	return &remoteNode{
		config:     config,
		msgQueue:   make(chan *message.NodeWireMessage, 1024),
		done:		make(chan struct{}, 2),
		state:      nodeStateDisconnected,
		stateLock:	sync.Mutex{},
		parentNode: parent,
		logger:     logger,
		indexInParent: -1,
	}
}

func (r *remoteNode) setState(state remoteNodeState)  {
	r.stateLock.Lock()
	defer r.stateLock.Unlock()

	r.state = state
}

func (r *remoteNode) setConnection(conn *comms.Connection)  {
	r.connection = conn
}


//set up the pinging and response go routine
func (r *remoteNode) startPinging()  {
	r.pingTimer = time.NewTicker(time.Second * time.Duration(r.config.PingInterval))
	r.pingTimeout = time.NewTimer(time.Second * time.Duration(r.config.PingTimeout))

	r.sendMessage(&message.PingMessage{}) //send the first ping message

	go func() {
		done := false
		for {
			select {
			case <-r.pingTimer.C:
				r.sendMessage(&message.PingMessage{})
				r.pingTimeout.Stop()
				r.pingTimeout.Reset(time.Second * time.Duration(r.config.PingTimeout))
			case <-r.done:		//we have this so that the goroutine would not linger after this node disconnects because
				done = true		//of the blocking channel in the above case statement
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
			case <-r.done:		//we have this so that the goroutine would not linger after this node disconnects because
				done = true		//of the blocking channel in the above case statement
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

func (r *remoteNode) start()  {
	go r.networkConsumer()
	go r.handleMessage()
	r.sendVerify()
	r.startPinging()
}

//join a cluster. this will be called if join in the config is set to true
func (r *remoteNode) join() error {
	utils.Info(r.logger, "joining node via " + r.config.IpAddress )

	go func() {				//goroutine will try to connect to the cluster until it succeeds
							//TODO: set an upper limit to the tries
		var err error
		tries := 0
		for {
			if err = r.connect(); err == nil {
				break
			}
			utils.Error(r.logger, err.Error())
			time.Sleep(time.Second * 3)
			tries++
			if r.config.ConnectRetries > 0 {
				if tries >= r.config.ConnectRetries {
					utils.Warn(r.logger, fmt.Sprintf("unable to connect to remote node '%s' after max retires", r.config.IpAddress))
					return
				}
			}
		}
		utils.Info(r.logger, "connected to node via " + r.config.IpAddress)
		r.start()
	}()


	return nil
}

//handles to low level connection to remote node
func (r *remoteNode) connect() error {
	var err error
	utils.Info(r.logger, "connecting to " + r.config.IpAddress)
	r.connection, err = comms.NewConnection(r.config.IpAddress, time.Second * 5)
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
	data := make([]byte, 4 + len(msg.Data))
	binary.LittleEndian.PutUint16(data, uint16(len(msg.Data) + 2))  //the 2 is for the message code
	binary.LittleEndian.PutUint16(data[2:],msg.Code)
	copy(data[4:], msg.Data)
	if err := r.connection.SendData(data); err != nil {
		utils.Critical(r.logger,"unexpected error while sending data [" + err.Error() + "]")
		r.shutDown()
	}
}


//bring down the remote node. should not be called from outside networkConsumer()
func (r *remoteNode) shutDown()  {
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
	utils.Info(r.logger, fmt.Sprintf("shutting down remote node '%s' ",r.config.Id))
	//r.logger = nil
}

//just queue the message in a channel
func (r *remoteNode) queueMessage(msg *message.NodeWireMessage) {

	if r.state == nodeStateHandshake {	//when in the handshake state only accept MsgVERIFY and MsgVERIFYRsp messages
		code := msg.Code
		if code != message.MsgVERIFY {
			return
		}
	}

	r.msgQueue <- msg
}


//message handler
func (r *remoteNode) handleMessage()  {

	for msg := range r.msgQueue {
		switch msg.Code {
		case message.MsgVERIFY:
			r.handleVerify(msg)
		case message.MsgPING:
			r.handlePing()
		case message.MsgPONG:
			r.handlePong()
		case message.MsgNODELIST:
			r.handleNodeList(msg)
		}
	}

	utils.Info(r.logger,fmt.Sprintf("terminated message handler goroutine for '%s'", r.config.Id))
}

func (r *remoteNode) sendVerify() {
	verifyMsgRsp := message.VerifyMessage{Id:r.parentNode.config.Id,
						ServicePort:strconv.Itoa(r.parentNode.config.LocalPort)}
	r.sendMessage(&verifyMsgRsp)
}

func (r *remoteNode) handleVerify(msg *message.NodeWireMessage) {

	verifyMsgRsp := message.VerifyMessage{}
	verifyMsgRsp.DeSerialize(msg)
	r.config.Id = verifyMsgRsp.Id
	r.config.ServicePort = verifyMsgRsp.ServicePort
	if !r.parentNode.eventVerifyRemoteNode(r) {
		utils.Warn(r.logger, fmt.Sprintf("node already has remote node '%s' so shutdown new connection", r.config.Id))
		r.shutDown()
		return
	}

	r.setState(nodeStateConnected)
	r.sendNodeList()
}

func (r *remoteNode) handlePing() {
	r.sendMessage(&message.PongMessage{})
}

func (r *remoteNode) handlePong() {
	r.pingTimeout.Stop()						//stop the timer since we got a response
	atomic.StoreInt32(&r.pingFailure, 0)		//reset failure counter since we got a response
}

func (r *remoteNode) sendNodeList() {
	values := r.parentNode.getRemoteNodes()
	length := len(values) - 1
	nodeList := make([]message.ProposedPeer, length, length)
	x := 0
	for _, v := range values {
		n := v.(*remoteNode)
		if n.config.Id == r.config.Id {
			continue
		}
		host, _, _ := net.SplitHostPort(n.config.IpAddress)
		nodeList[x] = message.ProposedPeer{Id:n.config.Id, IpAddress:net.JoinHostPort(host,n.config.ServicePort)}
		x++
	}

	if len(nodeList) > 0 {
		r.sendMessage(&message.NodeListMessage{List: nodeList})
	}
}

func (r *remoteNode) handleNodeList(msg *message.NodeWireMessage) {
	listMsg := message.NodeListMessage{}
	listMsg.DeSerialize(msg)
	length := len(listMsg.List)
	for x := 0; x < length; x++ {
		r.parentNode.joinQueue <- &listMsg.List[x]
	}
}



