package ringbuffer

import (
	"testing"
)

var rb RingBuffer[uint64]

// ensure the pushes worked
func TestPush(t *testing.T) {
    rb = NewRingBuffer[uint64](5)
    rb.Push(10)
    rb.Push(20)
    rb.Push(30)
    rb.Push(40)
    rb.Push(50)
    expected := 5
    result := rb.GetSize()

    if result != expected {
        t.Errorf("Push 5 elements. Size = %d; expected %d", result, expected)
    }
}

func TestMaxSize(t *testing.T) {
    rb = NewRingBuffer[uint64](5)
    rb.Push(10)
    rb.Push(20)
    rb.Push(30)
    rb.Push(40)
    rb.Push(50)
    expected := 5
    result := rb.MaxSize()

    if result != expected {
        t.Errorf("MaxSize. Size = %d; expected %d", result, expected)
    }
}

func TestPeak(t *testing.T) {
    rb = NewRingBuffer[uint64](5)
    rb.Push(10)
    rb.Push(20)
    rb.Push(30)
    rb.Push(40)
    rb.Push(50)
    expected := uint64(10)
    result, isEmpty := rb.Peek()

    if result != expected {
        t.Errorf("Peak. Size = %d; expected %d", result, expected)
    }
    if isEmpty {
        t.Errorf("isEmpty returned true on full ringbuffer.")
    }
}

func TestPop(t *testing.T) {
    rb = NewRingBuffer[uint64](5)
    rb.Push(10)
    rb.Push(20)
    rb.Push(30)
    rb.Push(40)
    rb.Push(50)
    expected := uint64(10)
    result, isEmpty := rb.Pop()

    if result != expected {
        t.Errorf("Pop. Size = %d; expected %d", result, expected)
    }
    if isEmpty {
        t.Errorf("isEmpty returned true on full ringbuffer.")
    }
}

// new ring buffer of strings, overwrite
func TestOverwrite(t *testing.T) {
    r := NewRingBuffer[string](3)
    r.Push("this")
    r.Push("is")
    r.Push("a")
    r.Push("test")
   
    strings := r.GetAll()
    result := strings[0]
    expected := "is"

    if result != expected {
        t.Errorf("Overwrite. result = %s; expected %s", result, expected)
    }

    if !r.IsFull {
        t.Errorf("Overwrite. isFull= %v; expected %v", true, false)
    }
}

func TestGetAll(t *testing.T) {
    r := NewRingBuffer[int](3)
    r.Push(1)
    r.Push(2)
    r.Push(3)
    r.Push(4)
    r.Push(5)

    ints := r.GetAll()

    expected := make([]int, 3)
    expected[0] = 3
    expected[1] = 4
    expected[2] = 5

    for i := range ints {
        if expected[i] != ints[i]{
            t.Errorf("Overwrite. result = %v; expected %v", ints, expected)
        }
    }
}

