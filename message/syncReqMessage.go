package message

type SyncReqMessage struct {
	code		uint16
}

func (sc *SyncReqMessage) Serialize() *NodeWireMessage {
	return &NodeWireMessage{Code: MsgSyncReq}
}

func (sc *SyncReqMessage) DeSerialize(msg *NodeWireMessage)  {
	sc.code = msg.Code
}
