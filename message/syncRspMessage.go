package message

import "encoding/json"

//SyncRspMessage is the struct that represents the response to a sync message
type SyncRspMessage struct {
	Code              uint64         `json:"code"`
	ReplicationFactor int            `json:"replication_factor"`
	List              []ProposedPeer `json:"list"`
}

//Serialize sync response message to node wire message
func (sc *SyncRspMessage) Serialize() *NodeWireMessage {
	sc.Code = MsgSyncRsp
	data, _ := json.Marshal(sc)
	return &NodeWireMessage{Code: MsgSyncRsp, Data: data}
}

//DeSerialize node wire message into sync response message
func (sc *SyncRspMessage) DeSerialize(msg *NodeWireMessage) {
	json.Unmarshal(msg.Data, sc)
}
