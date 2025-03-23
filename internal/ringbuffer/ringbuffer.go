package ringbuffer

type RingBuffer[T any] struct {
    Data []T
    Size int
    Head int
    Tail int
    IsFull bool
}

func NewRingBuffer[T any](size int) RingBuffer[T] {
    return RingBuffer[T]{
        //data: make([]T, size),
        Data: make([]T, size, size),
        Size: size,
        Head: 0,
        Tail: 0,
        IsFull: false,
    }
}

// Add item to ring buffer.  If full, will overwrite oldest element
func (r *RingBuffer[T]) Push(item T) {
    // move tail when buffer is full
    if r.IsFull {
        r.Tail = (r.Tail + 1) % r.Size
    }

    // insert at head position
    r.Data[r.Head] = item
    // move head, wrapping if required
    r.Head = (r.Head + 1) % r.Size

    // test for full
    if r.Head == r.Tail {
        r.IsFull = true
    }
}

/*******************************************************************
** returns oldest element from ring buff and removes it from buffer
** isEmpty returns true if ring is empty
********************************************************************/
func (r *RingBuffer[T]) Pop() (T, bool) {
    // If buffer empty, return a zero value of T and false
    if r.Head == r.Tail && !r.IsFull {
        var zeroValue T
        return zeroValue, true 
    }

    value := r.Data[r.Tail]
    r.Tail = (r.Tail + 1) % r.Size
    r.IsFull = false 

    return value, false 
}

/*******************************************************************
** return oldest element from ring buff without removing from buffer
** isEmpty returns true if ring is empty
********************************************************************/
func (r *RingBuffer[T]) Peek() (item T, isEmpty bool) {
    // If buffer empty, return a zero value of T and false
    if r.Head == r.Tail && !r.IsFull {
        var zeroValue T
        return zeroValue, true 
    }

    value := r.Data[r.Tail]
    return value, false 
}

// return all element in order for things like range loops
func (r *RingBuffer[T]) GetAll() []T {
    // empty case
    if !r.IsFull && r.Head == r.Tail {
        return []T{}
    }

    result := make([]T, 0, r.Size)

    if r.IsFull {
        for i:=0; i<r.Size; i++ {
            index := (r.Tail + i) % r.Size
            result = append(result, r.Data[index])
        }
    } else {
        if r.Head > r.Tail {
            result = append(result, r.Data[r.Tail:r.Head]...)
        } else {
            result = append(result, r.Data[r.Tail:]...)
            result = append(result, r.Data[r.Head:]...)
        }
    }

    return result
}

// get number of elements filled
func (r *RingBuffer[T]) GetSize() int {
    if r.IsFull {
        return r.Size
    }

    // not full, but equal is empty
    if r.Head == r.Tail {
        return 0
    }

    if r.Head > r.Tail {
        return r.Head - r.Tail
    }

    return r.Size - (r.Tail - r.Head)
}

// get number of max num of elements
func (r *RingBuffer[T]) MaxSize() int {
    return r.Size
}
