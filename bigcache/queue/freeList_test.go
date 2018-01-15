package queue

import (
	"math/rand"
	"testing"
)

func TestFreeList_add(t *testing.T) {
	list := newFreeList()
	list.add(0, 63)
	if len(list.sizeList[0]) != 1 {
		t.Error("sizeList index 0 should be 1 in length")
	}

	list.add(1024, 600)
	if len(list.sizeList[4]) != 1 {
		t.Error("sizeList index 4 should be 1 in length")
	}

	list.add(63, 100)
	if len(list.sizeList[0]) > 0 {
		t.Error("sizeList index 0 should not be 1 in length")
	}

}

func TestFreeList_find(t *testing.T) {
	list := newFreeList()
	list.add(0, 63)
	if len(list.sizeList[0]) != 1 {
		t.Error("sizeList index 0 should be 1 in length")
	}

	list.add(1024, 600)
	if len(list.sizeList[4]) != 1 {
		t.Error("sizeList index 4 should be 1 in length")
	}

	list.add(63, 100)
	if len(list.sizeList[0]) > 0 {
		t.Error("sizeList index 0 should not be 1 in length")
	}

	idx := list.find(1024 * 1024)
	if idx != -1 {
		t.Error("there should not be big enough room for this request")
	}

	t.Log(list.sizeList)
	idx = list.find(500)
	t.Log(idx)
	t.Log(list.sizeList)
	t.Log(list.sizeList[1][0])
}

var gList *freeList

func init() {
	gList = newFreeList()
	for x := 0; x < 1024; x++ {
		gList.add(x, rand.Intn(1024*1024*1024))
	}
}

func BenchmarkFreelist(b *testing.B) {

	for x := 0; x < b.N; x++ {
		gList.find(rand.Intn(1024 * 1024))
	}
}
