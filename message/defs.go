package message

const (
	MsgVERIFY = iota + 10
	MsgVERIFYRsp
	MsgPING
	MsgPONG
	MsgPUT
	MsgDEL
)

type NodeWireMessage struct {
	Code	uint16
	Data 	[]byte
}

type NodeMessage interface {
	Serialize() *NodeWireMessage
	DeSerialize(msg *NodeWireMessage)
}

