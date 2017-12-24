package message

import "encoding/json"

type SyncReqMessage struct {
	Code uint16		`json:"code"`
	Mode byte		`json:"mode"`
}

func (sc *SyncReqMessage) Serialize() *NodeWireMessage {
	data, _ := json.Marshal(sc)
	return &NodeWireMessage{Code: MsgSyncReq, Data: data}
}

func (sc *SyncReqMessage) DeSerialize(msg *NodeWireMessage) {
	json.Unmarshal(msg.Data, sc)
}
