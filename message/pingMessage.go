package message

type PingMessage struct {
	code uint16
}

func (pm *PingMessage) Serialize() *NodeWireMessage {
	return &NodeWireMessage{Code: MsgPING}
}

func (pm *PingMessage) DeSerialize(msg *NodeWireMessage) {
	pm.code = msg.Code
}
