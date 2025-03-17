package ringbuffer

type RingBuffer[T any] struct {
    data []T
    size int
    head int
    tail int
    isFull bool
}

func NewRingBuffer[T any](size int) RingBuffer[T] {
    return RingBuffer[T]{
        //data: make([]T, size),
        data: make([]T, size, size),
        size: size,
        head: 0,
        tail: 0,
        isFull: false,
    }
}

// Add item to ring buffer.  If full, will overwrite oldest element
func (r *RingBuffer[T]) Push(item T) {
    // move tail when buffer is full
    if r.isFull {
        r.tail = (r.tail + 1) % r.size
    }

    // insert at head position
    r.data[r.head] = item
    // move head, wrapping if required
    r.head = (r.head + 1) % r.size

    // test for full
    if r.head == r.tail {
        r.isFull = true
    }
}

/*******************************************************************
** returns oldest element from ring buff and removes it from buffer
** isEmpty returns true if ring is empty
********************************************************************/
func (r *RingBuffer[T]) Pop() (T, bool) {
    // If buffer empty, return a zero value of T and false
    if r.head == r.tail && !r.isFull {
        var zeroValue T
        return zeroValue, true 
    }

    value := r.data[r.tail]
    r.tail = (r.tail + 1) % r.size
    r.isFull = false 

    return value, false 
}

/*******************************************************************
** return oldest element from ring buff without removing from buffer
** isEmpty returns true if ring is empty
********************************************************************/
func (r *RingBuffer[T]) Peek() (item T, isEmpty bool) {
    // If buffer empty, return a zero value of T and false
    if r.head == r.tail && !r.isFull {
        var zeroValue T
        return zeroValue, true 
    }

    value := r.data[r.tail]
    return value, false 
}

// return all element in order for things like range loops
func (r *RingBuffer[T]) GetAll() []T {
    // empty case
    if !r.isFull && r.head == r.tail {
        return []T{}
    }

    result := make([]T, 0, r.size)

    if r.isFull {
        for i:=0; i<r.size; i++ {
            index := (r.tail + i) % r.size
            result = append(result, r.data[index])
        }
    } else {
        if r.head > r.tail {
            result = append(result, r.data[r.tail:r.head]...)
        } else {
            result = append(result, r.data[r.tail:]...)
            result = append(result, r.data[r.head:]...)
        }
    }

    return result
}

// get number of elements filled
func (r *RingBuffer[T]) Size() int {
    if r.isFull {
        return r.size
    }

    // not full, but equal is empty
    if r.head == r.tail {
        return 0
    }

    if r.head > r.tail {
        return r.head - r.tail
    }

    return r.size - (r.tail - r.head)
}

// get number of max num of elements
func (r *RingBuffer[T]) MaxSize() int {
    return r.size
}
