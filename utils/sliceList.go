package utils

import (
	"sync"
	"sync/atomic"
)

//SliceList represents a list of slices
type SliceList struct {
	items *sync.Map
	size  int32
}

//NewSliceList creates a new list
func NewSliceList() *SliceList {
	return &SliceList{size: 0, items: &sync.Map{}}
}

// Add appends a value at the end of the list
func (list *SliceList) Add(key, value interface{}) {
	if nil == value {
		return
	}
	atomic.AddInt32(&list.size, 1)
	list.items.Store(key, value)
}

//Contains verifies if a key exist within the list
func (list *SliceList) Contains(key interface{}) bool {
	_, ok := list.items.Load(key)
	return ok
}

//Size returns the size of the list
func (list *SliceList) Size() int32 {
	return list.size
}

// Remove removes one or more elements from the list with the supplied indices.
func (list *SliceList) Remove(key interface{}) {
	if _, ok := list.items.Load(key); ok {
		list.items.Delete(key)
		atomic.AddInt32(&list.size, -1)
	}
}

//Get returns the corresponding values of a key and true if it exists within the list
func (list *SliceList) Get(key interface{}) (interface{}, bool) {
	return list.items.Load(key)
}

//Values returns the values of every key in the list
func (list *SliceList) Values() []interface{} {
	newElements := make([]interface{}, 0, list.size)
	list.items.Range(func(key, value interface{}) bool {
		if value != nil {
			newElements = append(newElements, value)
		}
		return true
	})
	return newElements
}

//Keys returns the keys within the list
func (list *SliceList) Keys() *sync.Map {
	return list.items
}
