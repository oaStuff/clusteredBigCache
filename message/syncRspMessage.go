package message

import "encoding/json"

type SyncRspMessage struct {
	Code              uint64         `json:"code"`
	ReplicationFactor int            `json:"replication_factor"`
	List              []ProposedPeer `json:"list"`
}

func (sc *SyncRspMessage) Serialize() *NodeWireMessage {
	sc.Code = MsgSyncRsp
	data, _ := json.Marshal(sc)
	return &NodeWireMessage{Code: MsgSyncRsp, Data: data}
}

func (sc *SyncRspMessage) DeSerialize(msg *NodeWireMessage) {
	json.Unmarshal(msg.Data, sc)
}
