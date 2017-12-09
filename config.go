package clusteredBigCache

import "github.com/oaStuff/clusteredBigCache/cluster"

type ClusterConfig struct {
	cluster.NodeConfig
}

func DefaultClusterConfig() *ClusterConfig {
	return &ClusterConfig{
		NodeConfig: cluster.NodeConfig{
			Join:           false,
			BindAll:        true,
			LocalPort:      9911,
			ConnectRetries: 5,
			TerminateOnListenerExit: false,
			ReplicationFactor: 1,
			WriteAck:          true,
		},
	}
}
