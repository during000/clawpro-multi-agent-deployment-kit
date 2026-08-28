package common

import "testing"

func TestFilter(t *testing.T) {
	got := Filter([]uint{0, 3, 0, 2}, func(v uint) bool { return v != 0 })
	want := []uint{3, 2}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestFilterEmpty(t *testing.T) {
	got := Filter([]uint{}, func(v uint) bool { return v != 0 })
	if got == nil {
		t.Fatal("empty result should be non-nil")
	}
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}

func TestUnique(t *testing.T) {
	got := Unique([]uint{3, 1, 3, 2, 1})
	want := []uint{3, 1, 2}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestUniqueBy(t *testing.T) {
	type item struct {
		ID   uint
		Name string
	}
	got := UniqueBy([]item{
		{ID: 1, Name: "first"},
		{ID: 2, Name: "second"},
		{ID: 1, Name: "duplicate"},
	}, func(v item) uint { return v.ID })
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (%v)", len(got), got)
	}
	if got[0].Name != "first" || got[1].Name != "second" {
		t.Fatalf("got %v, want first occurrence order", got)
	}
}

func TestUniqueEmpty(t *testing.T) {
	got := Unique([]string{})
	if got == nil {
		t.Fatal("empty result should be non-nil")
	}
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}
