package test

import (
	"bytes"
	"github.com/oaStuff/clusteredBigCache/bigcache"
	"testing"
	"time"
)

func TestReplacingSameKey(t *testing.T) {
	bc, _ := bigcache.NewBigCache(bigcache.DefaultConfig())
	bc.Set("one", []byte("one"), time.Second*5)
	bc.Set("one", []byte("three"), time.Second*5)
	val, _ := bc.Get("one")
	if !bytes.Equal(val, []byte("three")) {
		t.Error("returned value ought to be equal ot 'three'")
	}
}

func TestReplacingSameKeyWithDiffExpiry(t *testing.T) {
	bc, _ := bigcache.NewBigCache(bigcache.DefaultConfig())
	bc.Set("one", []byte("one"), time.Second*5)
	bc.Set("one", []byte("three"), time.Second*3)
	val, _ := bc.Get("one")
	if !bytes.Equal(val, []byte("three")) {
		t.Error("returned value ought to be equal ot 'three'")
	}
	time.Sleep(time.Second * 4)
	val, _ = bc.Get("one")
	if nil != val {
		t.Error("value ought to have expired after 3 seconds")
	}
}

func TestRemove(t *testing.T) {
	bc, _ := bigcache.NewBigCache(bigcache.DefaultConfig())
	bc.Set("one", []byte("one"), time.Second*5)
	bc.Delete("one")
	val, _ := bc.Get("one")
	if nil != val {
		t.Error("value ought not to still be in cache after been deleted")
	}
}

func TestExpiry(t *testing.T) {
	bc, _ := bigcache.NewBigCache(bigcache.DefaultConfig())
	bc.Set("one", []byte("one"), time.Second*5)
	time.Sleep(time.Second * 3)
	val, _ := bc.Get("one")
	if !bytes.Equal(val, []byte("one")) {
		t.Error("returned value ought to be equal 'one'")
	}
	time.Sleep(time.Second * 3)
	val, _ = bc.Get("one")
	if nil != val {
		t.Error("value ought to have expired")
	}
}
