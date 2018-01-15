package message

import (
	"encoding/binary"
)

//PutMessage is message struct for sending data across to a remoteNode
type PutMessage struct {
	Code   uint16 `json:"code"`
	Key    string `json:"key"`
	Expiry uint64 `json:"expiry"`
	Data   []byte `json:"data"`
}

//Serialize put message to node wire message
func (pm *PutMessage) Serialize() *NodeWireMessage {
	msg := &NodeWireMessage{Code: MsgPUT}
	bKey := []byte(pm.Key)
	keyLen := len(bKey)
	msg.Data = make([]byte, keyLen+len(pm.Data)+2+8) //2 is needed for the size of the key while 8 is for expiry
	binary.LittleEndian.PutUint64(msg.Data, pm.Expiry)
	binary.LittleEndian.PutUint16(msg.Data[8:], uint16(keyLen))
	copy(msg.Data[(8+2):], bKey)
	copy(msg.Data[(8+2+keyLen):], pm.Data)

	return msg
}

//DeSerialize node wire message into put message
func (pm *PutMessage) DeSerialize(msg *NodeWireMessage) {
	pm.Code = MsgPUT
	pm.Expiry = binary.LittleEndian.Uint64(msg.Data)
	keyLen := binary.LittleEndian.Uint16(msg.Data[8:])
	pm.Key = string(msg.Data[(8 + 2):(8 + 2 + keyLen)])
	pm.Data = msg.Data[(8 + 2 + keyLen):]
}
