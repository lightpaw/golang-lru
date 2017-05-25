package simplelru

import "testing"

type testValue struct {
	i int64
	v uint64
}

func (t *testValue) Version() uint64 {
	return t.v
}

func TestLRU(t *testing.T) {
	evictCounter := 0
	onEvicted := func(k int64, v VersionedValue) {
		if k != v.(*testValue).i {
			t.Fatalf("Evict values not equal (%v!=%v)", k, v)
		}
		evictCounter += 1
	}
	l, err := NewLRU(128, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	for i := 0; i < 256; i++ {
		l.Add(int64(i), &testValue{i: int64(i)})
	}
	if l.Len() != 128 {
		t.Fatalf("bad len: %v", l.Len())
	}

	if evictCounter != 128 {
		t.Fatalf("bad evict count: %v", evictCounter)
	}

	for i, k := range l.Keys() {
		if v, ok := l.Get(k); !ok || v.(*testValue).i != k || v.(*testValue).i != int64(i+128) {
			t.Fatalf("bad key: %v", k)
		}
	}
	for i := 0; i < 128; i++ {
		_, ok := l.Get(int64(i))
		if ok {
			t.Fatalf("should be evicted")
		}
	}
	for i := 128; i < 256; i++ {
		_, ok := l.Get(int64(i))
		if !ok {
			t.Fatalf("should not be evicted")
		}
	}
	for i := 128; i < 192; i++ {
		ok := l.Remove(int64(i))
		if !ok {
			t.Fatalf("should be contained")
		}
		ok = l.Remove(int64(i))
		if ok {
			t.Fatalf("should not be contained")
		}
		_, ok = l.Get(int64(i))
		if ok {
			t.Fatalf("should be deleted")
		}
	}

	l.Get(192) // expect 192 to be last key in l.Keys()

	for i, k := range l.Keys() {
		if (i < 63 && k != int64(i+193)) || (i == 63 && k != 192) {
			t.Fatalf("out of order key: %v", k)
		}
	}

	l.Purge()
	if l.Len() != 0 {
		t.Fatalf("bad len: %v", l.Len())
	}
	if _, ok := l.Get(200); ok {
		t.Fatalf("should contain nothing")
	}
}

func TestLRU_GetOldest_RemoveOldest(t *testing.T) {
	l, err := NewLRU(128, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	for i := 0; i < 256; i++ {
		l.Add(int64(i), &testValue{i: int64(i)})
	}
	k, _, ok := l.GetOldest()
	if !ok {
		t.Fatalf("missing")
	}
	if k != 128 {
		t.Fatalf("bad: %v", k)
	}

	k, _, ok = l.RemoveOldest()
	if !ok {
		t.Fatalf("missing")
	}
	if k != 128 {
		t.Fatalf("bad: %v", k)
	}

	k, _, ok = l.RemoveOldest()
	if !ok {
		t.Fatalf("missing")
	}
	if k != 129 {
		t.Fatalf("bad: %v", k)
	}
}

// Test that Contains doesn't update recent-ness
func TestLRU_Contains(t *testing.T) {
	l, err := NewLRU(2, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	l.Add(1, &testValue{i: 1})
	l.Add(2, &testValue{i: 2})
	if !l.Contains(1) {
		t.Errorf("1 should be contained")
	}

	l.Add(3, &testValue{i: 3})
	if l.Contains(1) {
		t.Errorf("Contains should not have updated recent-ness of 1")
	}
}

// Test that Peek doesn't update recent-ness
func TestLRU_Peek(t *testing.T) {
	l, err := NewLRU(2, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	l.Add(1, &testValue{i: 1})
	l.Add(2, &testValue{i: 2})
	if v, ok := l.Peek(1); !ok || v.(*testValue).i != 1 {
		t.Errorf("1 should be set to 1: %v, %v", v, ok)
	}

	l.Add(3, &testValue{i: 3})
	if l.Contains(1) {
		t.Errorf("should not have updated recent-ness of 1")
	}
}

func TestLRU_Add_WithVersion(t *testing.T) {
	l, err := NewLRU(2, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if v := l.Add(1, &testValue{i: 1, v: 10}); v.Version() != 10 {
		t.Fatalf("should return obj")
	}
	l.Add(2, &testValue{i: 2, v: 10})

	if v := l.Add(1, &testValue{i: 11, v: 9}); v.Version() != 10 {
		t.Fatalf("should not return old version")
	}

	if v, ok := l.Peek(1); !ok || v.(*testValue).i != 1 {
		t.Errorf("1 should be set to 1: %v, %v", v, ok)
	}

	if v := l.Add(1, &testValue{i: 22, v: 11}); v.Version() != 11 {
		t.Fatalf("should return new version")
	}

	if v, ok := l.Peek(1); !ok || v.(*testValue).i != 22 {
		t.Errorf("1 should be set to 22: %v, %v", v, ok)
	}

	if !l.Contains(2) {
		t.Errorf("should not have updated recent-ness of 1")
	}

}
