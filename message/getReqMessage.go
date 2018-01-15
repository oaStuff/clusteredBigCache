package message

import "encoding/json"

type GetReqMessage struct {
	Code       uint16 `json:"code"`
	Key        string `json:"key"`
	PendingKey string `json:"pending_key"`
}

func (gm *GetReqMessage) Serialize() *NodeWireMessage {
	gm.Code = MsgGETReq
	data, _ := json.Marshal(gm)
	return &NodeWireMessage{Code: MsgGETReq, Data: data}
}

func (gm *GetReqMessage) DeSerialize(msg *NodeWireMessage) {
	json.Unmarshal(msg.Data, gm)
}
