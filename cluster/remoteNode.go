package cluster

import (
	"github.com/oaStuff/clusteredBigCache/comms"
	"time"
	"github.com/oaStuff/clusteredBigCache/utils"
	"encoding/binary"
	"io"
	"github.com/oaStuff/clusteredBigCache/message"
	"fmt"
	"sync/atomic"
)

const (
	nodeStateConnecting         = 	iota
	nodeStateConnected
	nodeStateDisconnected
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
	pingTimer			*time.Ticker		//used to send ping message to remote
	pingTimeout			*time.Timer			//used to monitor ping response
	pingFailure			int32				//count the number of pings without response
}

func newRemoteNode(config *remoteNodeConfig, parent *Node, logger utils.AppLogger) *remoteNode {
	config.PingFailureThreshHold = -1
	return &remoteNode{
		config:     config,
		msgQueue:   make(chan *message.NodeWireMessage, 1024),
		state:      nodeStateDisconnected,
		parentNode: parent,
		logger:     logger,
		indexInParent: -1,
	}
}

func (r *remoteNode) setState(state remoteNodeState)  {
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
		for range r.pingTimer.C {
			r.sendMessage(&message.PingMessage{})
			r.pingTimeout.Stop()
			r.pingTimeout.Reset(time.Second * time.Duration(r.config.PingTimeout))
		}
	}()

	go func() {
		for range r.pingTimeout.C {
			utils.Warn(r.logger,
				fmt.Sprintf("no ping response within configured time frame for remote node '%s'", r.config.Id))
			atomic.AddInt32(&r.pingFailure, 1)
			if r.pingFailure >= r.config.PingFailureThreshHold {
				r.pingTimeout.Stop()
				break
			}
		}
		//the remote node is assumed to be 'dead' since it has not responded to recent ping request
		utils.Warn(r.logger, fmt.Sprintf("shutting down connection to remote node '%s'", r.config.Id))
		r.shutDown()
	}()
}

func (r *remoteNode) start()  {
	go r.networkConsumer()
	go r.handleMessage()
	r.sendMessage(&message.VerifyMessage{})
}

//join a cluster. this will be called if join in the config is set to true
func (r *remoteNode) join() error {
	utils.Info(r.logger, "Joining cluster via " + r.config.IpAddress )

	go func() {				//goroutine will try to connect to the cluster until it succeeds
		var err error		//TODO: set an upper limit to the tries
		for {
			if err = r.connect(); err == nil {
				break
			}
			utils.Error(r.logger, err.Error())
			time.Sleep(time.Second * 3)
		}
		r.start()
	}()


	return nil
}

//handles to low level connection to remote node
func (r *remoteNode) connect() error {
	var err error
	r.state = nodeStateConnecting
	utils.Info(r.logger, "connecting to " + r.config.IpAddress)
	r.connection, err = comms.NewConnection(r.config.IpAddress, time.Second * 5)
	if err != nil {
		return err
	}

	r.state = nodeStateHandshake
	return nil
}


//this is a goroutine dedicated in reading data fromt the network
//it does this by reading a 4bytes header which is
//	byte 1 & 2 == length of data
//	byte 3 & 4 == message code
//	the rest of the data based on length is the message body
func (r *remoteNode) networkConsumer() {

	for r.state == nodeStateConnected {
		header, err := r.connection.ReadData(4, 0)	//read 4 byte header
		if io.EOF == err {
			utils.Critical(r.logger, "remote node has disconnected")
			r.shutDown()
			return
		}

		dataLength := int16(binary.LittleEndian.Uint16(header)) - 2		//subtracted 2 becos of message code
		msgCode := binary.LittleEndian.Uint16(header[2:])
		if dataLength > 0 {
			data, err := r.connection.ReadData(uint(dataLength), 0)
			if io.EOF == err {
				utils.Critical(r.logger, "remote node has disconnected")
				r.shutDown()
				return
			}
			r.queueMessage(&message.NodeWireMessage{Code:msgCode, Data:data})	//queue message to be processed
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
	data := make([]byte, 4 + len(msg.Data))
	binary.LittleEndian.PutUint16(data, uint16(len(msg.Data) + 2))  //the 2 is for the message code
	binary.LittleEndian.PutUint16(data[2:],msg.Code)
	copy(data[4:], msg.Data)
	if err := r.connection.SendData(data); err != nil {
		utils.Critical(r.logger,"unexpected error while sending data [" + err.Error() + "]")
		r.shutDown()
	}
}

//bring down the remote node
func (r *remoteNode) shutDown()  {
	r.parentNode.eventRemoteNodeDisconneced(r)
	r.state = nodeStateDisconnected
	r.connection.Close()
	r.parentNode = nil
	r.logger = nil
	r.config = nil
	r.connection = nil
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

}

func (r *remoteNode) handlePing() {
	r.sendMessage(&message.PongMessage{})
}

func (r *remoteNode) handlePong() {
	r.pingTimeout.Stop()						//stop the timer since we got a response
	atomic.StoreInt32(&r.pingFailure, 0)		//reset failure counter since we got a response
}



