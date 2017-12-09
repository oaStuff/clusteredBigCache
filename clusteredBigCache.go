package clusteredBigCache

import (
	"github.com/oaStuff/clusteredBigCache/cluster"
	"github.com/oaStuff/clusteredBigCache/utils"
	"time"
)

type ClusteredBigCache struct {
	config  *ClusterConfig
	node    *cluster.Node
	started bool
	logger  utils.AppLogger
}

func New(config *ClusterConfig, logger utils.AppLogger) *ClusteredBigCache {
	return &ClusteredBigCache{config: config, logger: logger}
}

func (cbc *ClusteredBigCache) Start() error {

	cbc.checkConfig()
	cbc.node = cluster.NewNode(&cbc.config.NodeConfig, cbc.logger)
	cbc.node.Start()

	cbc.started = true
	return nil
}

func (cbc *ClusteredBigCache) ShutDown() {
	cbc.started = false
	cbc.node.ShutDown()
}

func (cbc *ClusteredBigCache) DoTest() {
	cbc.node.DoTest()
}

func (cbc *ClusteredBigCache) checkConfig() {
	if cbc.config.LocalPort < 1 {
		panic("Local port can not be zero.")
	}

	if cbc.config.ReplicationFactor < 1 {
		utils.Warn(cbc.logger, "Adjusting replication to 1 (no replication) because it was less than 1")
		cbc.config.ReplicationFactor = 1
	}
}

func (cbc *ClusteredBigCache) Set(key string, data []byte, duration time.Duration) error {
	return cbc.node.PutData(key, data, duration)
}


func (cbc *ClusteredBigCache) Get(key string) ([]byte, error) {
	return cbc.node.GetData(key)
}

func (cbc *ClusteredBigCache) Delete(key string) error {
	return cbc.node.DeleteData(key)
}

