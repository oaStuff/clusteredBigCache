package cluster

const (
	msgVERIFY = iota + 10
	msgPUT
	msgDEL
)

type nodeMessage struct {
	code	uint16
	data 	[]byte
}
