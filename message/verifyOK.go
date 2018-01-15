package message

//VerifyOKMessage is message for verification ok
type VerifyOKMessage struct {
}

//Serialize verify ok message to node wire message
func (vm *VerifyOKMessage) Serialize() *NodeWireMessage {
	return &NodeWireMessage{Code: MsgVERIFYOK, Data: nil}
}

//DeSerialize node wire message into verify ok message
func (vm *VerifyOKMessage) DeSerialize(msg *NodeWireMessage) {
}
