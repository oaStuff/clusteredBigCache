package utils

import (
	"testing"
)

func TestSliceList_Add(t *testing.T) {
	sl := NewSliceList()
	sl.Add(12, "value")
	if sl.Contains(121) {
		t.Error("value should not be in list")
	}

	if !sl.Contains(12) {
		t.Error("value should be in the list")
	}
}

func TestSlice_Index(t *testing.T) {
	sl := NewSliceList()
	sl.Add(0, "value")
	if data, ok := sl.Get(0); !ok && "value" != data {
		t.Error("indexing not working properly")
	}
}

func TestSliceList_Remove(t *testing.T) {
	sl := NewSliceList()
	sl.Add(12, 12)
	sl.Add(44, 44)
	sl.Add(123, 123)

	sl.Remove(12)
	if sl.Contains(12) {
		t.Error("value ought to have been deleted")
	}
}

func TestSliceList_Size(t *testing.T) {
	sl := NewSliceList()
	sl.Add(12, 12)
	if sl.size != 1 {
		t.Error("size of list should be one")
	}

	sl.Add(123, 123)
	sl.Add(909, 909)
	sl.Add(834, 834)
	sl.Add(76123, 76123)

	if sl.size != 5 {
		t.Error("size should be equal to five")
	}

	sl.Remove(9091)
	if sl.size != 5 {
		t.Error("size should be equal to five")
	}

	sl.Remove(909)
	if sl.size != 4 {
		t.Error("size should be equal to four")
	}
}
