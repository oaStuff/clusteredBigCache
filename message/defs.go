package message

const (
	MsgVERIFY = iota + 10
	MsgVERIFYOK
	MsgPING
	MsgPONG
	MsgPUT
	MsgDEL
	MsgSyncReq
	MsgSyncRsp
)

type NodeWireMessage struct {
	Code	uint16
	Data 	[]byte
}

type NodeMessage interface {
	Serialize() *NodeWireMessage
	DeSerialize(msg *NodeWireMessage)
}

type ProposedPeer struct {
	Id			string
	IpAddress	string
}

