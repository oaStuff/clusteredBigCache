package message

//PongMessage is the struct for a ping response (pong)
type PongMessage struct {
	code uint16
}

//Serialize pong message to node wire message
func (pm *PongMessage) Serialize() *NodeWireMessage {
	return &NodeWireMessage{Code: MsgPONG}
}

//DeSerialize node wire message into pong message
func (pm *PongMessage) DeSerialize(msg *NodeWireMessage) {
	pm.code = msg.Code
}
