clusteredBigCache
=================

This is a library based on [bigcache](https://github.com/allegro/bigcache) with some modifications to support
* clustering and
* individual item expiration

Bigcache is an excellent piece of software but the fact that items could only expire based on a predefined 
value was not just too appealing. Bigcache had to be modified to support individual expiration of items using
a single timer. This happens by you specifying a time value as you add items to the cache.
Running two or more instances of an application that would require some level of caching would normally
default to memcache or redis which are external application, adding to the mix of services required for your
application to run.

With clusteredBigCache there is no requirement to run an external application to provide caching for multiple
instances of your application. The library handles caching as well as clustering the caches between multiple
instances of your application providing you with simple library APIs (by just calling functions) to store and
get your values.

With clusteredBigCache, when you store a value in one instance of your application every other instance 
or any other application for that matter that you configure to form/join your "cluster" will
see that exact same value.

## Installing

### Using *go get*

    $ go get github.com/oaStuff/clusteredBigCache

## Sample

Let's have a look at an example 
```go
package main

import (

    "os"
    "os/signal"
    "syscall"
    "fmt"
    "github.com/oaStuff/clusteredBigCache/Cluster"
    "flag"
    "time"
)
    

func main() {

    join := flag.String("join", "", "ipAddr:port number of remote server")
    replicationFactor := flag.Int("rf", 1, "replication factor for data stored")
    localPort := flag.Int("port", 6060, "local server port to bind to")
    debugPort := flag.Int("dport", 0, "debug port for debugging")
    
    flag.Parse()
    config := clusteredBigCache.DefaultClusterConfig()
    config.ReplicationFactor = *replicationFactor
    if *join != "" {
        config.JoinIp = *join
        config.Join = true
    }
    config.LocalPort = *localPort
    if *debugPort != 0 {
        config.DebugPort = *debugPort
        config.DebugMode = true
    }
    
    cache := clusteredBigCache.New(config, nil)
    
    
    cache.Start()
    
    
    
    
    
    c := make(chan os.Signal, 1)
    signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
    fmt.Println("press ctrl + C to exit")
    <-c
    fmt.Println("program stopped")


}


```