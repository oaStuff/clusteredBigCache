package message

type VerifyOKMessage struct {
}

func (vm *VerifyOKMessage) Serialize() *NodeWireMessage {
	return &NodeWireMessage{Code: MsgVERIFYOK, Data: nil}
}

func (vm *VerifyOKMessage) DeSerialize(msg *NodeWireMessage) {
}
