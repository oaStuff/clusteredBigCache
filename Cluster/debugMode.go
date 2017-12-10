package clusteredBigCache

import (
	"github.com/gin-gonic/gin"
	"net"
	"strconv"
	"net/http"
	"time"
)

func (node *ClusteredBigCache) startUpHttpServer()  {
	if node.config.DebugPort < 1024 {
		panic("debug port must be greater than 1024")
	}

	g := gin.Default()
	g.GET("/local_get/:key", func(c *gin.Context) {
		key := c.Param("key")
		data, ok := node.cache.Get(key)
		c.JSON(http.StatusOK, map[string]string{"key1":key, "exist": strconv.FormatBool(ok == nil),	"data": string(data)})
	})

	g.POST("/local_put/:key/:data", func(c *gin.Context) {
		key := c.Param("key")
		data := c.Param("data")
		node.cache.Set(key, []byte(data), time.Hour * 3)
		c.String(http.StatusOK, "successfully set ", key)
	})

	g.Run(net.JoinHostPort("",strconv.Itoa(node.config.DebugPort)))
}
