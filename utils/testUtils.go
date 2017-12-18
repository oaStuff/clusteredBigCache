package utils

import (
	"net"
	"strconv"
	"github.com/oaStuff/clusteredBigCache/message"
	"encoding/binary"

	"time"
)

type TestClient struct {
	conn *net.TCPConn
	respondToPings	bool
}

type TestServer struct {
	TestClient
	port			int
	started			chan struct{}
}

func NewTestClient() *TestClient {
	return &TestClient{}
}

func NewTestServer(port int, respondToPings bool) *TestServer {
	return &TestServer{port: port,
	TestClient: TestClient{respondToPings: respondToPings},
	started: make(chan struct{})}
}

func (s *TestServer) Start() error {
	l, err := net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(s.port)))
	if err != nil {
		return err
	}
	go func() {
		conn, _ := l.Accept()
		s.conn = conn.(*net.TCPConn)
		go s.readNetwork()
		close(s.started)
		l.Close()
	}()

	return nil
}

func (c *TestClient) Close()  {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *TestServer) SendVerifyMessage(id string)  {
	<-c.started
	verifyMsg := message.VerifyMessage{Id: id, ServicePort: "1111"}
	wMsg := verifyMsg.Serialize()
	d := make([]byte, 6 + len(wMsg.Data))
	binary.LittleEndian.PutUint32(d, uint32(len(wMsg.Data) + 2))  //the 2 is for the message code
	binary.LittleEndian.PutUint16(d[4:],wMsg.Code)
	copy(d[6:], wMsg.Data)
	c.conn.Write(d)
}

func (c *TestClient) readNetwork()  {

	go func() {
		for range time.NewTicker(time.Second * 1).C {
			msg := &message.PingMessage{}
			wireMsg := msg.Serialize()
			data := make([]byte, 6 + len(wireMsg.Data))
			binary.LittleEndian.PutUint32(data, uint32(len(wireMsg.Data) + 2))  //the 2 is for the message code
			binary.LittleEndian.PutUint16(data[4:],wireMsg.Code)
			copy(data[6:], wireMsg.Data)
			c.conn.Write(data)
		}
	}()

	header := make([]byte, 6)
	for {
		_, err := c.conn.Read(header)
		if err != nil {
			break
		}

		length := binary.LittleEndian.Uint32(header)
		code := binary.LittleEndian.Uint16(header[4:])

		var data []byte = nil
		if (length - 2) > 0 {
			data = make([]byte, length-2)
			_, err = c.conn.Read(data)
			if err != nil {
				break
			}
		}

		if code == message.MsgVERIFY {
			msg := &message.VerifyOKMessage{}
			wireMsg := msg.Serialize()
			data := make([]byte, 6 + len(wireMsg.Data))
			binary.LittleEndian.PutUint32(data, uint32(len(wireMsg.Data) + 2))  //the 2 is for the message code
			binary.LittleEndian.PutUint16(data[4:],wireMsg.Code)
			copy(data[6:], wireMsg.Data)
			c.conn.Write(data)

		}else if code == message.MsgPING {
			if c.respondToPings {
				msg := &message.PongMessage{}
				wireMsg := msg.Serialize()
				data := make([]byte, 6+len(wireMsg.Data))
				binary.LittleEndian.PutUint32(data, uint32(len(wireMsg.Data)+2)) //the 2 is for the message code
				binary.LittleEndian.PutUint16(data[4:], wireMsg.Code)
				copy(data[6:], wireMsg.Data)
				c.conn.Write(data)
			}

		} else if code == message.MsgVERIFYOK  {
			msg := &message.SyncReqMessage{}
			wireMsg := msg.Serialize()
			data := make([]byte, 6 + len(wireMsg.Data))
			binary.LittleEndian.PutUint32(data, uint32(len(wireMsg.Data) + 2))  //the 2 is for the message code
			binary.LittleEndian.PutUint16(data[4:],wireMsg.Code)
			copy(data[6:], wireMsg.Data)
			c.conn.Write(data)

		} else if code == message.MsgPONG {


		} else if code == message.MsgSyncReq {
			listMsg := message.SyncRspMessage{}
			listMsg.List = []message.ProposedPeer{{Id:"remote_1", IpAddress:"192.168.56.21:9990"},
				{Id:"remote_2", IpAddress:"172.16.111.89:9991"},
				{Id:"remote_3", IpAddress:"10.10.0.1:9090"}}

			wMsg := listMsg.Serialize()
			d := make([]byte, 6 + len(wMsg.Data))
			binary.LittleEndian.PutUint32(d, uint32(len(wMsg.Data) + 2))  //the 2 is for the message code
			binary.LittleEndian.PutUint16(d[4:],wMsg.Code)
			copy(d[6:], wMsg.Data)
			c.conn.Write(d)
		}
	}
}