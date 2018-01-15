package message

//PingMessage is the message struct for sending ping messages
type PingMessage struct {
	code uint16
}

//Serialize ping message to node wire message
func (pm *PingMessage) Serialize() *NodeWireMessage {
	return &NodeWireMessage{Code: MsgPING}
}

//DeSerialize node wire message into ping message
func (pm *PingMessage) DeSerialize(msg *NodeWireMessage) {
	pm.code = msg.Code
}
