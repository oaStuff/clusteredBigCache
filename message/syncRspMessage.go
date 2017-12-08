package message

import "encoding/json"

type SyncRspMessage struct {
	Code uint16         `json:"code"`
	List []ProposedPeer `json:"list"`
}

func (sc *SyncRspMessage) Serialize() *NodeWireMessage {
	data, _ := json.Marshal(sc)
	return &NodeWireMessage{Code: MsgSyncRsp, Data: data}
}

func (sc *SyncRspMessage) DeSerialize(msg *NodeWireMessage) {
	json.Unmarshal(msg.Data, sc)
}