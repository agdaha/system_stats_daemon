package ringbuffer_test

import (
	"testing"
	"time"

	"system_stats_deamon/internal/ringbuffer"
)

func TestPushAndSince(t *testing.T) {
	rb := ringbuffer.New[int](10)
	before := time.Now()
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	entries := rb.Since(before)
	if len(entries) != 3 {
		t.Errorf("want 3 entries, got %d", len(entries))
	}
}

func TestSince_Empty(t *testing.T) {
	rb := ringbuffer.New[int](10)
	entries := rb.Since(time.Now())
	if len(entries) != 0 {
		t.Errorf("want 0 entries from empty buffer, got %d", len(entries))
	}
}

func TestOverflow_KeepsNewest(t *testing.T) {
	rb := ringbuffer.New[int](3)
	before := time.Now()
	for i := range 5 {
		rb.Push(i)
	}

	entries := rb.Since(before)
	if len(entries) != 3 {
		t.Fatalf("want 3 entries after overflow, got %d", len(entries))
	}
	if entries[0].Data != 2 {
		t.Errorf("want oldest entry = 2, got %d", entries[0].Data)
	}
}

func TestSince_Cutoff(t *testing.T) {
	rb := ringbuffer.New[int](10)
	rb.Push(1)
	rb.Push(2)

	cutoff := time.Now()
	time.Sleep(time.Millisecond)
	rb.Push(3)

	entries := rb.Since(cutoff)
	if len(entries) != 1 {
		t.Fatalf("want 1 entry after cutoff, got %d", len(entries))
	}
	if entries[0].Data != 3 {
		t.Errorf("want entry 3, got %d", entries[0].Data)
	}
}

func TestOrder_IsInsertion(t *testing.T) {
	rb := ringbuffer.New[int](5)
	before := time.Now()
	for i := range 5 {
		rb.Push(i)
	}

	entries := rb.Since(before)
	for i, e := range entries {
		if e.Data != i {
			t.Errorf("entry[%d]: want %d, got %d", i, i, e.Data)
		}
	}
}
