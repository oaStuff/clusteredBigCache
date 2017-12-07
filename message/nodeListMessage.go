package message

import "encoding/json"

type NodeListMessage struct {
	List 		[]ProposedPeer		`json:"list"`
}

func (lr *NodeListMessage) Serialize() *NodeWireMessage {
	data, _ := json.Marshal(lr)
	return &NodeWireMessage{Code:MsgNODELIST, Data:data}
}

func (lr *NodeListMessage) DeSerialize(msg *NodeWireMessage)  {
	json.Unmarshal(msg.Data, lr)
}

