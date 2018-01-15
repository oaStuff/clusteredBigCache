package queue

import (
	"github.com/emirpasic/gods/trees/avltree"
	"github.com/emirpasic/gods/utils"
)

//item that will trace free space in the parent slice
type itemPos struct {
	actualSize  int
	parentIndex int
}

type posArray []*itemPos

//freelist data structure
type freeList struct {
	sizeList  [8]posArray
	indexTree *avltree.Tree
}

//create a new freeList
func newFreeList() *freeList {
	fl := &freeList{
		indexTree: avltree.NewWith(utils.IntComparator),
	}

	for x := 0; x < 8; x++ {
		fl.sizeList[x] = make(posArray, 0, 256)
	}

	return fl
}

//add is to add an index of a parent slice along with its size into the freelist.
//this is done by checking if previous items in the freelist is adjacent to the new one been added
//if so combine them into a single entry
func (list *freeList) add(index, size int) error {
	//NOTE: every added entry will create 2 entries in the indexTree
	//this is to help identify its start and end positions.

	item := &itemPos{parentIndex: index, actualSize: size}
	if d, found := list.indexTree.Get(index - 1); found { //check if there is an adjacent entry to the 'left' of index
		v := d.(*itemPos)
		list.indexTree.Remove(index - 1)
		list.indexTree.Remove(v.parentIndex)
		list.removeFromSizeList(v)

		item.parentIndex = v.parentIndex
		item.actualSize = v.actualSize + size
	}
	if d, found := list.indexTree.Get(index + size); found { //check if there is an adjacent entry to the 'right' of index
		v := d.(*itemPos)
		list.indexTree.Remove(index + size)
		list.indexTree.Remove(v.parentIndex + v.actualSize - 1)
		list.removeFromSizeList(v)

		item.parentIndex = index
		item.actualSize = size + v.actualSize
	}

	//store it into the indexTree. both its start and end position
	list.indexTree.Put(item.parentIndex, item)
	list.indexTree.Put(item.parentIndex+item.actualSize-1, item)
	list.putIntoSizeList(item)

	return nil
}

func (list *freeList) removeFromSizeList(v *itemPos) {
	var pos int
	switch {
	case v.actualSize < 64:
		pos = listPos(v, list.sizeList[0])
		if pos != -1 {
			copy(list.sizeList[0][pos:], list.sizeList[0][(pos+1):])
			list.sizeList[0] = list.sizeList[0][:len(list.sizeList[0])-1]
		}
	case v.actualSize < 128:
		pos = listPos(v, list.sizeList[1])
		if pos != -1 {
			copy(list.sizeList[1][pos:], list.sizeList[1][(pos+1):])
			list.sizeList[1] = list.sizeList[1][:len(list.sizeList[1])-1]
		}
	case v.actualSize < 256:
		pos = listPos(v, list.sizeList[2])
		if pos != -1 {
			copy(list.sizeList[2][pos:], list.sizeList[2][(pos+1):])
			list.sizeList[2] = list.sizeList[2][:len(list.sizeList[2])-1]
		}
	case v.actualSize < 512:
		pos = listPos(v, list.sizeList[3])
		if pos != -1 {
			copy(list.sizeList[3][pos:], list.sizeList[3][(pos+1):])
			list.sizeList[3] = list.sizeList[3][:len(list.sizeList[3])-1]
		}
	case v.actualSize < 1024:
		pos = listPos(v, list.sizeList[4])
		if pos != -1 {
			copy(list.sizeList[4][pos:], list.sizeList[4][(pos+1):])
			list.sizeList[4] = list.sizeList[4][:len(list.sizeList[4])-1]
		}
	case v.actualSize < 2048:
		pos = listPos(v, list.sizeList[5])
		if pos != -1 {
			copy(list.sizeList[5][pos:], list.sizeList[5][(pos+1):])
			list.sizeList[5] = list.sizeList[5][:len(list.sizeList[5])-1]
		}
	case v.actualSize < 4096:
		pos = listPos(v, list.sizeList[6])
		if pos != -1 {
			copy(list.sizeList[6][pos:], list.sizeList[6][(pos+1):])
			list.sizeList[6] = list.sizeList[6][:len(list.sizeList[6])-1]
		}
	case v.actualSize > 4096:
		pos = listPos(v, list.sizeList[7])
		if pos != -1 {
			copy(list.sizeList[7][pos:], list.sizeList[7][(pos+1):])
			list.sizeList[7] = list.sizeList[7][:len(list.sizeList[7])-1]
		}
	}
}

func (list *freeList) putIntoSizeList(item *itemPos) {
	switch {
	case item.actualSize < 64:
		list.sizeList[0] = append(list.sizeList[0], item)
	case item.actualSize < 128:
		list.sizeList[1] = append(list.sizeList[1], item)
	case item.actualSize < 256:
		list.sizeList[2] = append(list.sizeList[2], item)
	case item.actualSize < 512:
		list.sizeList[3] = append(list.sizeList[3], item)
	case item.actualSize < 1024:
		list.sizeList[4] = append(list.sizeList[4], item)
	case item.actualSize < 2048:
		list.sizeList[5] = append(list.sizeList[5], item)
	case item.actualSize < 4096:
		list.sizeList[6] = append(list.sizeList[6], item)
	case item.actualSize > 4096:
		list.sizeList[7] = append(list.sizeList[7], item)
	}
}

func listPos(v *itemPos, arr posArray) int {
	for x := 0; x < len(arr); x++ {
		if v == arr[x] {
			return x
		}
	}

	return -1
}

func (list *freeList) find(size int) int {

	index := 0
	buckSize := 0
	switch {
	case size < 64:
		index = 0
		buckSize = 64
	case size < 128:
		index = 1
		buckSize = 128
	case size < 256:
		index = 2
		buckSize = 256
	case size < 512:
		index = 3
		buckSize = 512
	case size < 1024:
		index = 4
		buckSize = 1024
	case size < 2048:
		index = 5
		buckSize = 2048
	case size < 4096:
		index = 6
		buckSize = 4096
	case size > 4096:
		index = 7
		buckSize = 4097
	}

	for idx := index; idx < 8; idx++ {
		if v, found := list.findInSizeList(idx, size, buckSize); found {
			return v
		}
		if idx < 6 {
			buckSize *= 2
		} else {
			buckSize = 4097
		}

	}

	return -1
}

func (list *freeList) findInSizeList(idx int, size int, buckSize int) (int, bool) {
	for x := 0; x < len(list.sizeList[idx]); x++ {
		tmp := list.sizeList[idx][x]
		if size <= tmp.actualSize {
			list.indexTree.Remove(tmp.parentIndex)
			if size == tmp.actualSize {
				copy(list.sizeList[idx][x:], list.sizeList[idx][(x+1):])
				list.sizeList[idx] = list.sizeList[idx][:len(list.sizeList[idx])-1]
				list.indexTree.Remove(tmp.actualSize + tmp.parentIndex - 1)
				return tmp.parentIndex, true
			}

			if (tmp.actualSize - size) >= buckSize {
				ret := tmp.parentIndex
				tmp.parentIndex += size
				tmp.actualSize -= size
				list.indexTree.Put(tmp.parentIndex, tmp)
				return ret, true
			}

			{
				list.indexTree.Remove(tmp.parentIndex + tmp.actualSize - 1)
				copy(list.sizeList[idx][x:], list.sizeList[idx][(x+1):])
				list.sizeList[idx] = list.sizeList[idx][:len(list.sizeList[idx])-1]
				list.add(tmp.parentIndex+size, tmp.actualSize-size)
				return tmp.parentIndex, true
			}
		}
	}

	return -1, false
}
