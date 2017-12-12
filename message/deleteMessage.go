package message

import "encoding/json"

type DeleteMessage struct {
	Code	uint16		`json:"code"`
	Key		string		`json:"key"`
}

func (dm *DeleteMessage) Serialize() *NodeWireMessage {
	dm.Code = MsgDEL
	data, _ := json.Marshal(dm)
	return &NodeWireMessage{Code: MsgDEL, Data: data}
}

func (dm *DeleteMessage) DeSerialize(msg *NodeWireMessage) {
	json.Unmarshal(msg.Data, dm)
}

