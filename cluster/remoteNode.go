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
	Id        				string 		`json:"id"`
	IpAddress 				string 		`json:"ip_address"`
	PingFailureThreshHold	int32		`json:"ping_failure_thresh_hold"`
	PingInterval			int 		`json:"ping_interval"`
	PingTimeout				int 		`json:"ping_timeout"`
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
	return item1.(*remoteNode).config.Id == item2.(*remoteNode).config.Id
}

func checkConfig(logger utils.AppLogger, config *remoteNodeConfig)  {
	config.PingFailureThreshHold = -1
	if config.PingInterval < 5 {
		config.PingInterval = 15
	}

	if config.PingTimeout < 3 {
		config.PingTimeout = 13
	}

	if config.PingTimeout > config.PingInterval {
		utils.Warn(logger, "ping timeout is greater than ping interval, pings will NEVER timeout")
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

	go func() {
		val := -1
		for {
			fmt.Println("select")
			select {
			case <-r.pingTimer.C:
				val = 1
				fmt.Println("set to 1")
			case <-r.done:
				val = 2
				fmt.Println("set to 2")
			}
			if val == 1 {
				r.sendMessage(&message.PingMessage{})
				r.pingTimeout.Stop()
				r.pingTimeout.Reset(time.Second * time.Duration(r.config.PingTimeout))
			}else if val == 2 {
				break
			}
			val = -1
		}
		fmt.Println("clooooooosssseed")
	}()

	go func() {
		val := -1
		for {
			fmt.Println("select 2")
			select {
			case <-r.pingTimeout.C:
				val = 1
			case <-r.done:
				val = 2
				fmt.Println("vvvvvvvvvvvvvvvvvvvvvvvvvvvvvv")
			}
			fmt.Println("outsidddddddeeeeee")
			if val == 1 {
				utils.Warn(r.logger,
					fmt.Sprintf("no ping response within configured time frame for remote node '%s'", r.config.Id))
				atomic.AddInt32(&r.pingFailure, 1)
				if r.pingFailure >= r.config.PingFailureThreshHold {
					r.pingTimeout.Stop()
					break
				}
			}else if val == 2 {
				break
			}
			val = -1
		}

		if val != 2 {
			//the remote node is assumed to be 'dead' since it has not responded to recent ping request
			utils.Warn(r.logger, fmt.Sprintf("shutting down connection to remote node '%s' due to no ping response", r.config.Id))
			r.shutDown("form goroutine timer")
		}else {
			fmt.Println("yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy")
		}
	}()
}

func (r *remoteNode) start()  {
	go r.networkConsumer()
	go r.handleMessage()
	r.sendMessage(&message.VerifyMessage{})
}

//join a cluster. this will be called if join in the config is set to true
func (r *remoteNode) join() error {
	utils.Info(r.logger, "joining cluster via " + r.config.IpAddress )

	go func() {				//goroutine will try to connect to the cluster until it succeeds
		var err error		//TODO: set an upper limit to the tries
		for {
			if err = r.connect(); err == nil {
				break
			}
			utils.Error(r.logger, err.Error())
			time.Sleep(time.Second * 3)
		}
		utils.Info(r.logger, "connected to cluster via " + r.config.IpAddress)
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
			fmt.Println("+++++++++++++++++++++++=")
			utils.Critical(r.logger, "remote node has disconnected")
			r.shutDown("from networkConsumer()1")
			return
		}

		dataLength := int16(binary.LittleEndian.Uint16(header)) - 2 //subtracted 2 becos of message code
		msgCode := binary.LittleEndian.Uint16(header[2:])
		if dataLength > 0 {
			data, err := r.connection.ReadData(uint(dataLength), 0)
			if nil != err {
				utils.Critical(r.logger, "remote node has disconnected")
				r.shutDown("from networkConsumer()2")
				return
			}
			r.queueMessage(&message.NodeWireMessage{Code: msgCode, Data: data}) //queue message to be processed
		}
	}

}

//this sends a message on the network
//it builds the message using the following protocol
// bytes 1 & 2 == total length of the data (including the 2 byte message code)
// bytes 3 & 4 == message code
// bytes 5 upwards == message content
func (r *remoteNode) sendMessage(m message.NodeMessage) {
	msg := m.Serialize()
	fmt.Println("sending data === ", msg.Code)
	data := make([]byte, 4 + len(msg.Data))
	binary.LittleEndian.PutUint16(data, uint16(len(msg.Data) + 2))  //the 2 is for the message code
	binary.LittleEndian.PutUint16(data[2:],msg.Code)
	copy(data[4:], msg.Data)
	if err := r.connection.SendData(data); err != nil {
		utils.Critical(r.logger,"unexpected error while sending data [" + err.Error() + "]")
		r.shutDown("from sending message")
	}
}


//bring down the remote node. should not be called from outside networkConsumer()
func (r *remoteNode) shutDown(d string)  {
	r.stateLock.Lock()
	defer r.stateLock.Unlock()

	fmt.Println("************************************88  ", d)
	if r.state == nodeStateDisconnected {
		fmt.Println("leaving shutdown early")
		return
	}

	r.state = nodeStateDisconnected
	r.parentNode.eventRemoteNodeDisconneced(r)
	r.connection.Close()
	r.parentNode = nil
	r.logger = nil
	r.pingTimeout.Stop()
	r.pingTimer.Stop()
	close(r.done)


	fmt.Println("leaving shutdown normarly")
}

//just queue the message in a channel
func (r *remoteNode) queueMessage(msg *message.NodeWireMessage) {

	if r.state == nodeStateHandshake {	//when is the handshake state only accept MsgVERIFY and MsgVERIFYRsp messages
		code := msg.Code
		if (code != message.MsgVERIFY) && (code != message.MsgVERIFYRsp) {
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
		case message.MsgVERIFYRsp:
			r.handleVerifyRsp(msg)
		case message.MsgPING:
			r.handlePing()
		case message.MsgPONG:
			r.handlePong()
		}
	}
}

func (r *remoteNode) handleVerify(msg *message.NodeWireMessage) {
	verifyMsg := message.VerifyMessage{}
	verifyMsg.DeSerialize(msg)
}

func (r *remoteNode) handleVerifyRsp(msg *message.NodeWireMessage) {
	utils.Info(r.logger, "verifying resp")
	verifyMsgRsp := message.VerifyMessageRsp{}
	verifyMsgRsp.DeSerialize(msg)
	if r.parentNode.eventVerifyRemoteNode(r, verifyMsgRsp) {
		r.config.Id = verifyMsgRsp.Id
		r.setState(nodeStateConnected)
		utils.Info(r.logger, "node changing state")
		r.startPinging()
	}
}

func (r *remoteNode) handlePing() {
	r.sendMessage(&message.PongMessage{})
}

func (r *remoteNode) handlePong() {
	r.pingTimeout.Stop()						//stop the timer since we got a response
	atomic.StoreInt32(&r.pingFailure, 0)		//reset failure counter since we got a response
}



