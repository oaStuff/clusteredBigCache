package clusteredBigCache

import (
	"github.com/oaStuff/clusteredBigCache/cluster"
	"github.com/oaStuff/clusteredBigCache/utils"
)

type ClusteredBigCache struct {
	config 		*ClusterConfig
	node 		*cluster.Node
	started		bool
	logger 		utils.AppLogger
}

func New(config *ClusterConfig, logger utils.AppLogger) *ClusteredBigCache {
	return &ClusteredBigCache{config:config, logger:logger}
}

func (cbc *ClusteredBigCache) Start() error {

	cbc.node = cluster.NewNode(&cbc.config.NodeConfig, cbc.logger)
	cbc.node.Start()

	cbc.started = true
	return nil
}