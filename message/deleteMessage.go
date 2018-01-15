package message

import "encoding/json"

//DeleteMessage is the struct message that defines which data to delete
type DeleteMessage struct {
	Code uint16 `json:"code"`
	Key  string `json:"key"`
}

//Serialize delete message to node wire message
func (dm *DeleteMessage) Serialize() *NodeWireMessage {
	dm.Code = MsgDEL
	data, _ := json.Marshal(dm)
	return &NodeWireMessage{Code: MsgDEL, Data: data}
}

//DeSerialize node wire message into delete message
func (dm *DeleteMessage) DeSerialize(msg *NodeWireMessage) {
	json.Unmarshal(msg.Data, dm)
}
