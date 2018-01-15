package bigcache

import (
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/emirpasic/gods/trees/avltree"
	"github.com/emirpasic/gods/utils"
	"sync"
	"time"
)

type ttlManager struct {
	shard       *cacheShard
	timeTree    *avltree.Tree
	treeLock    sync.Mutex
	timer       *time.Timer
	ShardHasher Hasher
}

func newTtlManager(shard *cacheShard, hasher Hasher) *ttlManager {
	ttl := &ttlManager{
		shard:       shard,
		timeTree:    avltree.NewWith(utils.UInt64Comparator),
		treeLock:    sync.Mutex{},
		timer:       time.NewTimer(time.Hour * 10), //fire the time on time. just for initialization
		ShardHasher: hasher,
	}

	go ttl.eviction()

	return ttl
}

// Put item in the tree with key = timestamp
// Manage collision with hashSet
func (ttl *ttlManager) put(timestamp uint64, cacheKey string) {

	ttl.treeLock.Lock()
	defer ttl.treeLock.Unlock()

	if data, found := ttl.timeTree.Get(timestamp); found { //a hashset already exist, add to it
		data.(*hashset.Set).Add(cacheKey)
		return

	}

	set := hashset.New() //new value, create a hashset and use it
	set.Add(cacheKey)
	ttl.timeTree.Put(timestamp, set)

	if ttl.timeTree.Size() == 1 {
		ttl.resetTimer(timestamp)
		return
	}

	//is this timestamp at the head, if so then it displaced the smallest one so reset timer
	if key, _ := ttl.peek(); key == timestamp {
		ttl.resetTimer(timestamp)
	}

}

//Remove a timestamp and key pair from the tree
func (ttl *ttlManager) remove(timestamp uint64, key string) {

	ttl.treeLock.Lock()
	defer ttl.treeLock.Unlock()

	if data, found := ttl.timeTree.Get(timestamp); found { // data is in hashset, only remove that one
		set := data.(*hashset.Set)
		set.Remove(key)
		if set.Empty() {
			if k, _ := ttl.peek(); k == timestamp {
				ttl.timeTree.Remove(timestamp)
				if k, _ = ttl.peek(); k != nil {
					ttl.resetTimer(k.(uint64))
				}
				return
			}
			ttl.timeTree.Remove(timestamp) //remove everything if empty
		}
	}

}

//reset everything to default
func (ttl *ttlManager) reset() {
	ttl.treeLock.Lock()
	ttl.timeTree.Clear()
	ttl.stopTimer()
	ttl.treeLock.Unlock()
}

//goroutine that handles eviction
func (ttl *ttlManager) eviction() {

	for range ttl.timer.C {

		ttl.treeLock.Lock()          //acquire tree lock
		if ttl.timeTree.Size() < 1 { //if tree is empty move on
			ttl.treeLock.Unlock()
			continue
		}

		key, value := ttl.peek() //peek the value at the head(the tree keys are ordered)
		ttl.timeTree.Remove(key)

		ttl.treeLock.Unlock()
		ttl.evict(key.(uint64), value)

		for { //loop through to ensure all expired keys are removed in this single step

			ttl.treeLock.Lock()
			if ttl.timeTree.Size() < 1 {
				break
			}

			key, value = ttl.peek()
			nextExpiryTime := key.(uint64)
			interval := nextExpiryTime - uint64(time.Now().Unix())
			if 0 == interval {
				ttl.timeTree.Remove(key)
				ttl.treeLock.Unlock()
				ttl.evict(key.(uint64), value)
			} else {
				ttl.resetTimer(nextExpiryTime) //TODO: should this not just call Reset() directly? seen that the timer just fired
				break                          // TODO: and would be in the expired state
			}
		}

		ttl.treeLock.Unlock()
	}
}

//Reset the eviction timer
func (ttl *ttlManager) resetTimer(timeStamp uint64) {

	interval := timeStamp - uint64(time.Now().Unix())
	if !ttl.timer.Stop() {
		select {
		case <-ttl.timer.C:
		default:
		}
	}
	ttl.timer.Reset(time.Second * time.Duration(interval))
}

//Stop the eviction timer
func (ttl *ttlManager) stopTimer() {
	if !ttl.timer.Stop() {
		select {
		case <-ttl.timer.C:
		default:
		}
	}
}

//Grab the data at the beginning of the iterator
func (ttl *ttlManager) peek() (interface{}, interface{}) {

	it := ttl.timeTree.Iterator()
	it.Next()
	return it.Key(), it.Value()
}

//Do the eviction from the tree and parent shard
func (ttl *ttlManager) evict(timeStamp uint64, value interface{}) {

	set := value.(*hashset.Set)
	ttl.shard.evictDel(timeStamp, set)
}
