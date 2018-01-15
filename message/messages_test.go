package message

import (
	"reflect"
	"testing"
	"time"
)

func TestPutMessage(t *testing.T) {
	msg := PutMessage{Code: MsgPUT, Expiry: uint64(time.Now().Unix()), Key: "key_1", Data: []byte("data_A")}
	newMsg := PutMessage{}
	newMsg.DeSerialize(msg.Serialize())
	if !reflect.DeepEqual(msg, newMsg) {
		t.Error("PutMessage serialization and deserialization not working properly")
	}
}

func TestSyncReqMessage(t *testing.T) {
	msg := SyncReqMessage{Code: MsgSyncReq, Mode: 20}
	newMsg := SyncReqMessage{}
	newMsg.DeSerialize(msg.Serialize())
	if !reflect.DeepEqual(msg, newMsg) {
		t.Error("SyncReqMessage serialization and deserialization not working properly")
	}
}

func TestSyncRspMessage(t *testing.T) {
	msg := SyncRspMessage{Code: MsgSyncRsp, List: []ProposedPeer{
		{Id: "id_1", IpAddress: "192.168.56.1"},
		{Id: "id_2", IpAddress: "192.168.56.2"},
		{Id: "id_3", IpAddress: "192.168.56.3"},
		{Id: "id_4", IpAddress: "192.168.56.4"},
		{Id: "id_5", IpAddress: "192.168.56.5"},
	}}

	newMsg := SyncRspMessage{}
	newMsg.DeSerialize(msg.Serialize())
	if !reflect.DeepEqual(msg, newMsg) {
		t.Error("SyncRspMessage serialization and deserialization not working properly")
	}
}

func TestVerifyMessage(t *testing.T) {
	msg := VerifyMessage{Id: "id_node", Version: "1.02", ServicePort: "9090"}
	newMsg := VerifyMessage{}
	newMsg.DeSerialize(msg.Serialize())
	if !reflect.DeepEqual(msg, newMsg) {
		t.Error("VerifyMessage serialization and deserialization not working properly")
	}
}

func TestDeleteMessage(t *testing.T) {
	msg := DeleteMessage{Key: "key_ab"}
	newMsg := DeleteMessage{}
	newMsg.DeSerialize(msg.Serialize())
	if !reflect.DeepEqual(msg, newMsg) {
		t.Error("DeleteMessage serialization and deserialization not working properly")
	}
}

func TestGetReqMessage(t *testing.T) {
	msg := GetReqMessage{Key: "key_ab", PendingKey: "pending_key_1"}
	newMsg := GetReqMessage{}
	newMsg.DeSerialize(msg.Serialize())
	if !reflect.DeepEqual(msg, newMsg) {
		t.Error("GetReqMessage serialization and deserialization not working properly")
	}
}

func TestGetRspMessage(t *testing.T) {
	msg := GetRspMessage{Code: MsgGETRsp, PendingKey: "pending_key_5", Data: []byte("data_5")}
	newMsg := GetRspMessage{}
	newMsg.DeSerialize(msg.Serialize())
	if !reflect.DeepEqual(msg, newMsg) {
		t.Error("GetRspMessage serialization and deserialization not working properly")
	}

	if newMsg.Data == nil {
		t.Log("data is nil")
	}
}
