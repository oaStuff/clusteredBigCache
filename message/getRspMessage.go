package message

import (
	"encoding/binary"
)

//GetRspMessage is the struct for the response message for a remote get
type GetRspMessage struct {
	Code       uint16 `json:"code"`
	PendingKey string `json:"pending_key"`
	Data       []byte `json:"data"`
}

//Serialize get response message to node wire message
func (gm *GetRspMessage) Serialize() *NodeWireMessage {
	msg := &NodeWireMessage{Code: MsgGETRsp}
	bKey := []byte(gm.PendingKey)
	keyLen := len(bKey)
	msg.Data = make([]byte, keyLen+len(gm.Data)+2) //2 is needed for the size of the key
	binary.LittleEndian.PutUint16(msg.Data, uint16(keyLen))
	copy(msg.Data[2:], bKey)
	copy(msg.Data[(2+keyLen):], gm.Data)

	return msg
}

//DeSerialize node wire message into get response message
func (gm *GetRspMessage) DeSerialize(msg *NodeWireMessage) {
	gm.Code = MsgGETRsp
	keyLen := binary.LittleEndian.Uint16(msg.Data)
	gm.PendingKey = string(msg.Data[2:(2 + keyLen)])
	gm.Data = msg.Data[(2 + keyLen):]
}
