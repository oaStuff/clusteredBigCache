package message

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

type NodeWireMessage struct {
	Code uint16
	Data []byte
}

type NodeMessage interface {
	Serialize() *NodeWireMessage
	DeSerialize(msg *NodeWireMessage)
}

type ProposedPeer struct {
	Id        string
	IpAddress string
}

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

	return "unknow"
}