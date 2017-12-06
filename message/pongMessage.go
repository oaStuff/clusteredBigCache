package message

type PongMessage struct {
	code		uint16
}

func (pm *PongMessage) Serialize() *NodeWireMessage {
	return &NodeWireMessage{Code:MsgPONG}
}

func (pm *PongMessage) DeSerialize(msg *NodeWireMessage)  {
	pm.code = msg.Code
}
