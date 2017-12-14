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

## Samples 1

This is the application responsible for storing data into the cache

```go
package main

import (
    "fmt"
    "bufio"
    "os"
    "github.com/oaStuff/clusteredBigCache/Cluster"
    "strings"
    "time"
)

//
//main function
//
func main() {
    fmt.Println("starting...")
    cache := clusteredBigCache.New(clusteredBigCache.DefaultClusterConfig(), nil)
    count := 1
    cache.Start()

    reader := bufio.NewReader(os.Stdin)
    var data string
    for strings.ToLower(data) != "exit" {
        fmt.Print("enter data : ")
        data, _ = reader.ReadString('\n')
        data = strings.TrimSpace(data)
        err := cache.Put(fmt.Sprintf("key_%d", count), []byte(data), time.Minute * 60)
        if err != nil {
            panic(err)
       	}
       	fmt.Printf("'%s' stored under key 'key_%d'\n", data, count)
       	count++
   	}
}

```

##### Explanation:

The above application captures data from the keyboard and stores them inside clusteredBigCache
start with keys 'key_1', 'key_2'...'key_n'. As the user types and presses the enter key the data is stored in 
the cache.

`cache := clusteredBigCache.New(clusteredBigCache.DefaultClusterConfig(), nil)`
This statement will create the cache using the default configuration. This configuration has default 
values for *LocalPort = 9911*, *Join = false* amongst others. If you intend to use this library for applications
that will run on the same machine, you will have to give unique values for *LocalPort*

`cache.Start()` This **must** be called before using any other method on this cache.

`err := cache.Put(fmt.Sprintf("key_%d", count), []byte(data), time.Minute * 60)`. You set values in the cache
giving it a key, the data as a `[]byte` slice and the expiration or time to live (ttl) for that key/value within the cache.
When the key/value pair reaches its expiration time, they are removed automatically.


## Samples 2

This is the application responsible for reading data of the cache. This can be run on the same or different machine
on the network.

```go
package main

import (
    "github.com/oaStuff/clusteredBigCache/Cluster"
    "bufio"
    "os"
    "strings"
    "fmt"
    "time"
)

//
//
func main() {
    config := clusteredBigCache.DefaultClusterConfig()
    config.LocalPort = 8888
    config.Join = true
    config.JoinIp = "127.0.0.1:9911"
    cache := clusteredBigCache.New(config, nil)
    err := cache.Start()
    if err != nil {
        panic(err)
    }
    
    reader := bufio.NewReader(os.Stdin)
    var data string
    for strings.ToLower(data) != "exit" {
        fmt.Print("enter key : ")
        data, _ = reader.ReadString('\n')
        data = strings.TrimSpace(data)
        value, err := cache.Get(data, time.Millisecond * 160)
        if err != nil {
            fmt.Println(err)
            continue
        }
        fmt.Printf("you got '%s' from the cache\n", value)
    }
}

```

##### Explanation:

The above application reads a string from the keyboard which should represent a key for a value in clusteredBigCache.
If a user enters the corresponding keys shown in sample1 above ('key_1', 'key_2'...'key_n'), the corresponding values
will be returned.

```go
    config := clusteredBigCache.DefaultClusterConfig()
    config.LocalPort = 8888
    config.Join = true
    config.JoinIp = "127.0.0.1:9911"
    cache := clusteredBigCache.New(config, nil)
    err := cache.Start()
```

The above uses the default configuration to create a config and modifies what is actually needed.
`config.LocalPort = 8888` has to be changed since this application will run on the same machine with the sample1 
application. This is to avoid 'port already in use' errors. 

`config.Join = true`. For an application to join another
application or applications using clusteredBigCache, it **must** set *config.Join* value to *true* and set `config.JoinIP` to 
the IP address of one of the systems using clusteredBigCache eg `config.Join = "127.0.0.1:9911`. This example says that this application 
wants to join to another application using clusteredBigCache at IP address *127.0.0.1* and port number *9911*.

`cache := clusteredBigCache.New(config, nil)` creates the cache and `cache.Start()` must be called to start everything running.

**NB**

After `cache.Start()` is called the library tries to connect to the specified IP address using the specified port. 
When successfully connected, it create a cluster of applications using clusteredBigCache as a single cache. ie all applications connected will see every value 
every application sets in the cache.





