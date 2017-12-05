package message

type VerifyMessage struct {
	code uint16
}

func (vm *VerifyMessage) Serialize() *NodeWireMessage  {
	return &NodeWireMessage{Code:MsgVERIFY}
}

func (vm *VerifyMessage) DeSerialize(msg *NodeWireMessage)  {
	vm.code = msg.Code
}