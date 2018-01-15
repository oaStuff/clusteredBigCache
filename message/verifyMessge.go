package message

import "encoding/json"

//VerifyMessage is message struct for node verification
type VerifyMessage struct {
	Id          string `json:"id"`
	Version     string `json:"version"`
	ServicePort string `json:"service_port"`
	Mode        byte   `json:"mode"`
}

//Serialize verify message to node wire message
func (vm *VerifyMessage) Serialize() *NodeWireMessage {
	data, _ := json.Marshal(vm)
	return &NodeWireMessage{Code: MsgVERIFY, Data: data}
}

//DeSerialize node wire message into verify message
func (vm *VerifyMessage) DeSerialize(msg *NodeWireMessage) {
	json.Unmarshal(msg.Data, vm)
}
