package message

import "encoding/json"

//GetReqMessage is message struct for getting request from remoteNode
type GetReqMessage struct {
	Code       uint16 `json:"code"`
	Key        string `json:"key"`
	PendingKey string `json:"pending_key"`
}

//Serialize get request message to node wire message
func (gm *GetReqMessage) Serialize() *NodeWireMessage {
	gm.Code = MsgGETReq
	data, _ := json.Marshal(gm)
	return &NodeWireMessage{Code: MsgGETReq, Data: data}
}

//DeSerialize node wire message into get request message
func (gm *GetReqMessage) DeSerialize(msg *NodeWireMessage) {
	json.Unmarshal(msg.Data, gm)
}
