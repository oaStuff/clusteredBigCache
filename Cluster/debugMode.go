package clusteredBigCache

import (
	"github.com/gin-gonic/gin"
	"net"
	"strconv"
	"net/http"
	"time"
	"fmt"
)

func (node *ClusteredBigCache) startUpHttpServer()  {
	if node.config.DebugPort < 1024 {
		panic("debug port must be greater than 1024")
	}

	g := gin.Default()
	g.GET("/local_get/:key", func(c *gin.Context) {
		key := c.Param("key")
		data, ok := node.cache.Get(key)
		c.JSON(http.StatusOK, map[string]string{"key":key, "exist": strconv.FormatBool(ok == nil),	"data": string(data)})
	})

	g.POST("/local_put/:key/:data", func(c *gin.Context) {
		key := c.Param("key")
		data := c.Param("data")
		node.cache.Set(key, []byte(data), time.Hour * 3)
		c.String(http.StatusOK, "successfully set ", key)
	})

	g.POST("/put/:key/:data/:time", func(c *gin.Context) {
		key := c.Param("key")
		data := c.Param("data")
		sec := c.Param("time")
		n, _ := strconv.Atoi(sec)
		node.Put(key, []byte(data), time.Minute * time.Duration(n))
	})

	g.GET("/get/:key", func(c *gin.Context) {
		key := c.Param("key")
		t1 := time.Now()
		data, ok := node.Get(key, time.Second * 1)
		t2 := time.Now().Sub(t1)
		fmt.Println("it took me ", t2)
		c.JSON(http.StatusOK, map[string]string{"key":key, "exist": strconv.FormatBool(ok == nil),	"data": string(data)})
	})

	g.Run(net.JoinHostPort("",strconv.Itoa(node.config.DebugPort)))
}
