package message

//Constants for message codes
const (
	MsgVERIFY = iota + 10
	MsgVERIFYOK
	MsgPING
	MsgPONG
	MsgPUT
	MsgGETReq
	MsgGETRsp
	MsgDEL
	MsgSyncReq
	MsgSyncRsp
)

//NodeWireMessage defines the struct that carries message on the wire
type NodeWireMessage struct {
	Code uint16
	Data []byte
}

//NodeMessage defines the interface for all message that can be sent and received
type NodeMessage interface {
	Serialize() *NodeWireMessage
	DeSerialize(msg *NodeWireMessage)
}

//ProposedPeer are used to represent peers that needs to be connected to
type ProposedPeer struct {
	Id        string
	IpAddress string
}

//MsgCodeToString maps message code to strings
func MsgCodeToString(code uint16) string {
	switch code {
	case MsgDEL:
		return "msgDelete"
	case MsgSyncRsp:
		return "msgSyncRsp"
	case MsgSyncReq:
		return "msgSyncReq"
	case MsgPUT:
		return "msgPUT"
	case MsgPONG:
		return "msgPONG"
	case MsgPING:
		return "mgsPING"
	case MsgVERIFYOK:
		return "msgVerifyOK"
	case MsgVERIFY:
		return "msgVerify"
	case MsgGETReq:
		return "msgGETReq"
	case MsgGETRsp:
		return "msgGETRsp"
	}

	return "unknown"
}
