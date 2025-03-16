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
        data: make([]T, size),
        size: size,
        head: 0,
        tail: 0,
        isFull: false,
    }
}

func (r *RingBuffer[T]) Push(item T) {
    if r.isFull {
        r.tail = (r.tail + 1) % r.size
    }

    r.data[r.head] = item
    r.head = (r.head + 1) % r.size

    if r.head == r.tail {
        r.isFull = true
    }
}

// remove oldest element from ring buff
func (r *RingBuffer[T]) Pop() (T, bool) {
    // If buffer empty, return a zero value of T and false
    if r.head == r.tail && !r.isFull {
        var zeroValue T
        return zeroValue, false
    }

    value := r.data[r.tail]
    r.tail = (r.tail + 1) % r.size
    r.isFull = false 

    return value, true
}

// return all element in order for things like range loops
func (r *RingBuffer[T]) GetAll() []T {
    var result []T
    if r.isFull {
        for i := r.tail; i != r.head; i = (i + 1) % r.size {
            result = append(result, r.data[i])
        }
    } else {
        for i := r.tail; i < r.head; i++ {
            result = append(result, r.data[i])
        }
    }
    return result
}

// get number of elements filled
func (r *RingBuffer[T]) Size() int {
    if r.isFull {
        return r.size
    }

    if (r.head == r.tail && r.isFull == false) {
        return 0;
    }

    if r.head >= r.tail {
        return r.head - r.tail
    }

    return r.size - (r.tail - r.head)
}

// get number of max num of elements
func (r *RingBuffer[T]) MaxSize() int {
    return r.size
}
