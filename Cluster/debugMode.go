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
	g.GET("/local/:key", func(c *gin.Context) {
		key := c.Param("key")
		data, ok := node.cache.Get(key)
		c.JSON(http.StatusOK, map[string]string{"key":key, "exist": strconv.FormatBool(ok == nil),	"data": string(data)})
	})

	g.GET("/global/:key", func(c *gin.Context) {
		key := c.Param("key")
		data, ok := node.Get(key, time.Second * 1)
		c.JSON(http.StatusOK, map[string]string{"key":key, "exist": strconv.FormatBool(ok == nil),	"data": string(data)})
	})

	g.POST("/local/:key/:data/:time", func(c *gin.Context) {
		key := c.Param("key")
		data := c.Param("data")
		n, _ := strconv.Atoi(c.Param("time"))
		node.cache.Set(key, []byte(data), time.Minute * time.Duration(n))
		c.String(http.StatusOK, "successfully set %s", key)
	})

	g.POST("/global/:key/:data/:time", func(c *gin.Context) {
		key := c.Param("key")
		data := c.Param("data")
		sec := c.Param("time")
		n, _ := strconv.Atoi(sec)
		node.Put(key, []byte(data), time.Minute * time.Duration(n))
		c.String(http.StatusOK, "successfully set %s", key)
	})

	g.DELETE("/local/:key", func(c *gin.Context) {
		key := c.Param("key")
		node.cache.Delete(key)
		c.String(http.StatusOK, "successfully deleted %s", key)
	})

	g.DELETE("/global/:key", func(c *gin.Context) {
		key := c.Param("key")
		node.Delete(key)
		c.String(http.StatusOK, "successfully deleted %s", key)
	})

	g.Run(net.JoinHostPort("",strconv.Itoa(node.config.DebugPort)))
}
