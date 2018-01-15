package message

import "encoding/json"

type VerifyMessage struct {
	Id          string `json:"id"`
	Version     string `json:"version"`
	ServicePort string `json:"service_port"`
	Mode        byte   `json:"mode"`
}

func (vm *VerifyMessage) Serialize() *NodeWireMessage {
	data, _ := json.Marshal(vm)
	return &NodeWireMessage{Code: MsgVERIFY, Data: data}
}

func (vm *VerifyMessage) DeSerialize(msg *NodeWireMessage) {
	json.Unmarshal(msg.Data, vm)
}
