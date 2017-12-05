package message

import "encoding/json"

type VerifyMessageRsp struct {
	Id		string		`json:"id"`
	version	string		`json:"version"`
}

func (vm *VerifyMessageRsp) Serialize() *NodeWireMessage {
	data, _ := json.Marshal(vm)
	return &NodeWireMessage{Code:MsgVERIFYRsp, Data:data}
}

func (vm *VerifyMessageRsp) DeSerialize(msg *NodeWireMessage)  {
	json.Unmarshal(msg.Data, vm)
}

