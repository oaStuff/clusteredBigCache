package message

import "encoding/json"

//SyncReqMessage is message struct to request synchronization from a remoteNode
type SyncReqMessage struct {
	Code uint16 `json:"code"`
	Mode byte   `json:"mode"`
}

//Serialize sync message to node wire message
func (sc *SyncReqMessage) Serialize() *NodeWireMessage {
	data, _ := json.Marshal(sc)
	return &NodeWireMessage{Code: MsgSyncReq, Data: data}
}

//DeSerialize node wire message into sync message
func (sc *SyncReqMessage) DeSerialize(msg *NodeWireMessage) {
	json.Unmarshal(msg.Data, sc)
}
