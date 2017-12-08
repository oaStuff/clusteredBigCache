package cluster

import (
	"net"
	"strconv"
	"github.com/oaStuff/clusteredBigCache/message"
	"encoding/binary"

	"time"
)

type testClient struct {
	conn *net.TCPConn
	respondToPings	bool
}

type testServer struct {
	testClient
	port			int
	started			chan struct{}
}

func newTestClient() *testClient {
	return &testClient{}
}

func newTestServer(port int, respondToPings bool) *testServer {
	return &testServer{port: port,
	testClient: testClient{respondToPings: respondToPings},
	started: make(chan struct{})}
}

func (s *testServer) start() error {
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

func (c *testClient) close()  {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *testServer) sendVerifyMessage(id string)  {
	<-c.started
	verifyMsg := message.VerifyMessage{Id: id, ServicePort: "1111"}
	wMsg := verifyMsg.Serialize()
	d := make([]byte, 4 + len(wMsg.Data))
	binary.LittleEndian.PutUint16(d, uint16(len(wMsg.Data) + 2))  //the 2 is for the message code
	binary.LittleEndian.PutUint16(d[2:],wMsg.Code)
	copy(d[4:], wMsg.Data)
	c.conn.Write(d)
}

func (c *testClient) readNetwork()  {

	go func() {
		for range time.NewTicker(time.Second * 1).C {
			msg := &message.PingMessage{}
			wireMsg := msg.Serialize()
			data := make([]byte, 4 + len(wireMsg.Data))
			binary.LittleEndian.PutUint16(data, uint16(len(wireMsg.Data) + 2))  //the 2 is for the message code
			binary.LittleEndian.PutUint16(data[2:],wireMsg.Code)
			copy(data[4:], wireMsg.Data)
			c.conn.Write(data)
		}
	}()

	header := make([]byte, 4)
	for {
		_, err := c.conn.Read(header)
		if err != nil {
			break
		}

		length := binary.LittleEndian.Uint16(header)
		code := binary.LittleEndian.Uint16(header[2:])

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
			data := make([]byte, 4 + len(wireMsg.Data))
			binary.LittleEndian.PutUint16(data, uint16(len(wireMsg.Data) + 2))  //the 2 is for the message code
			binary.LittleEndian.PutUint16(data[2:],wireMsg.Code)
			copy(data[4:], wireMsg.Data)
			c.conn.Write(data)

		}else if code == message.MsgPING {
			if c.respondToPings {
				msg := &message.PongMessage{}
				wireMsg := msg.Serialize()
				data := make([]byte, 4+len(wireMsg.Data))
				binary.LittleEndian.PutUint16(data, uint16(len(wireMsg.Data)+2)) //the 2 is for the message code
				binary.LittleEndian.PutUint16(data[2:], wireMsg.Code)
				copy(data[4:], wireMsg.Data)
				c.conn.Write(data)
			}

		} else if code == message.MsgVERIFYOK  {
			msg := &message.SyncReqMessage{}
			wireMsg := msg.Serialize()
			data := make([]byte, 4 + len(wireMsg.Data))
			binary.LittleEndian.PutUint16(data, uint16(len(wireMsg.Data) + 2))  //the 2 is for the message code
			binary.LittleEndian.PutUint16(data[2:],wireMsg.Code)
			copy(data[4:], wireMsg.Data)
			c.conn.Write(data)

		} else if code == message.MsgPONG {


		} else if code == message.MsgSyncReq {
			listMsg := message.SyncRspMessage{}
			listMsg.List = []message.ProposedPeer{{Id:"remote_1", IpAddress:"192.168.56.21:9990"},
				{Id:"remote_2", IpAddress:"172.16.111.89:9991"},
				{Id:"remote_3", IpAddress:"10.10.0.1:9090"}}

			wMsg := listMsg.Serialize()
			d := make([]byte, 4 + len(wMsg.Data))
			binary.LittleEndian.PutUint16(d, uint16(len(wMsg.Data) + 2))  //the 2 is for the message code
			binary.LittleEndian.PutUint16(d[2:],wMsg.Code)
			copy(d[4:], wMsg.Data)
			c.conn.Write(d)
		}
	}
}