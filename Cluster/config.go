package clusteredBigCache

type getReplyData struct {
	data	[]byte
}

type getRequestData struct {
	key			string
	randStr		string
	replyChan	chan *getReplyData
	done 		chan struct{}
}

type getRequestDataWrapper struct {
	g *getRequestData
	r *remoteNode
}

func DefaultClusterConfig() *ClusteredBigCacheConfig {

	return &ClusteredBigCacheConfig{
			Join:           false,
			BindAll:        true,
			LocalPort:      9911,
			ConnectRetries: 5,
			TerminateOnListenerExit: false,
			ReplicationFactor: 1,
			WriteAck:          true,
		}
}
