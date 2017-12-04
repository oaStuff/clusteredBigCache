package cluster

import (
	"github.com/oaStuff/clusteredBigCache/comms"
	"time"
	"github.com/oaStuff/clusteredBigCache/utils"
	"encoding/binary"
	"io"
)

const (
	nodeStateConnecting         = 	iota
	nodeStateConnected
	nodeStateDisconnected
	nodeStateHandshake
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
		config:     config,
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

func (r *remoteNode) join() error {
	utils.Info(r.logger, "Joining cluster via " + r.config.IpAddress )

	go func() {
		var err error
		for {
			if err = r.connect(); err == nil {
				break
			}
			utils.Error(r.logger, err.Error())
			time.Sleep(time.Second * 3)
		}
		go r.networkConsumer()
		r.sendMessage(&nodeMessage{code:msgVERIFY})
	}()


	return nil
}

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

func (r *remoteNode) networkConsumer() {

	for r.state == nodeStateConnected {
		header, err := r.connection.ReadData(4, 0)
		if io.EOF == err {
			utils.Critical(r.logger, "remote node has disconnected")
			r.shutDown()
			return
		}

		dataLength := int16(binary.LittleEndian.Uint16(header)) - 2
		msgCode := binary.LittleEndian.Uint16(header[2:])
		if dataLength > 0 {
			data, err := r.connection.ReadData(uint(dataLength), 0)
			if io.EOF == err {
				utils.Critical(r.logger, "remote node has disconnected")
				r.shutDown()
				return
			}
			go r.handleMessage(msgCode, data)
		}
	}
}

func (r *remoteNode) sendMessage(msg *nodeMessage) {
	data := make([]byte, 4 + len(msg.data))
	binary.LittleEndian.PutUint16(data, uint16(len(msg.data) + 2))
	binary.LittleEndian.PutUint16(data[2:],msg.code)
	copy(data[4:], msg.data)
	if err := r.connection.SendData(data); err != nil {
		utils.Critical(r.logger,"unexpected error while sending data [" + err.Error() + "]")
		r.shutDown()
	}
}

func (r *remoteNode) shutDown()  {
	r.parentNode.eventRemoteNodeDisconneced(r)
	r.state = nodeStateDisconnected
	r.connection.Close()
	r.parentNode = nil
	r.logger = nil
	r.config = nil
	r.connection = nil
}

func (r *remoteNode) handleMessage(code uint16, data []byte) {

}

