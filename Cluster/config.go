package clusteredBigCache


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
