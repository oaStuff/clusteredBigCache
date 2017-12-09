package utils

import (
	"testing"
	"strconv"
)

func equalFunc(item1, item2 interface{}) bool {
	if (nil == item1) || (nil == item2) {
		return false
	}
	return item1.(int) == item2.(int)
}

//used by SliceList to get the key of objects stored in the list
func keyFunc(item interface{}) string {
	if nil == item {
		return "0"
	}

	return strconv.Itoa(item.(int))
}

func TestSliceList_Add(t *testing.T) {
	sl := NewSliceList(equalFunc, keyFunc)
	sl.Add(12)
	if sl.Contains(121) {
		t.Error("value should not be in list")
	}

	if !sl.Contains(12) {
		t.Error("value should be in the list")
	}
}

func TestSlice_Index(t *testing.T)  {
	sl := NewSliceList(equalFunc, keyFunc)
	sl.Add(12)
	if _, ok := sl.Get(0); !ok {
		t.Error("indexing not working properly")
	}
}

func TestSliceList_Remove(t *testing.T)  {
	sl := NewSliceList(equalFunc, keyFunc)
	sl.Add(12)
	sl.Add(44)
	sl.Add(123)

	sl.Remove(1)
	if sl.Contains(44) {
		t.Error("value ought to have been deleted")
	}
}

func TestSliceList_Size(t *testing.T)  {
	sl := NewSliceList(equalFunc, keyFunc)
	sl.Add(12)
	if sl.size != 1 {
		t.Error("size of list should be one")
	}

	sl.Add(123)
	sl.Add(909)
	sl.Add(834)
	sl.Add(76123)

	if sl.size != 5 {
		t.Error("size should be equal to five")
	}

	sl.Remove(909)
	if sl.size != 5 {
		t.Error("size should be equal to five")
	}

	sl.Remove(0)
	if sl.size != 4 {
		t.Error("size should be equal to four")
	}
}
