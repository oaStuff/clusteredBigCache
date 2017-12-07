package message

import "encoding/json"

type NodeListRsp struct {
	List 		[]ProposedPeer		`json:"list"`
}

func (lr *NodeListRsp) Serialize() *NodeWireMessage {
	data, _ := json.Marshal(lr)
	return &NodeWireMessage{Code:MsgNODELIST, Data:data}
}

func (lr *NodeListRsp) DeSerialize(msg *NodeWireMessage)  {
	json.Unmarshal(msg.Data, lr)
}

