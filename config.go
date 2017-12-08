package clusteredBigCache

import "github.com/oaStuff/clusteredBigCache/cluster"

type ClusterConfig struct {
	cluster.NodeConfig
	ReplicationFactor	uint64		`json:"replication_factor"`
	WriteAck			bool		`json:"write_ack"`
}


func DefaultClusterConfig() *ClusterConfig {
	return &ClusterConfig{
		NodeConfig: cluster.NodeConfig{
			Join:				false,
			BindAll:			true,
			LocalPort:			9911,
			ConnectRetries: 	5,
		},
		ReplicationFactor:	1,
		WriteAck:			true,
	}
}
