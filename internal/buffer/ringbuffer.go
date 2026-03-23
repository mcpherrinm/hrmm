package buffer

import (
	"math"
	"sort"
	"time"
)

// RingBuffer stores a fixed number of float64 values in FIFO order.
// When capacity is reached, oldest values are overwritten.
type RingBuffer struct {
	data     []float64
	capacity int
	head     int // next write position
	size     int // current number of elements
}

// New creates a new RingBuffer with the specified capacity.
func New(capacity int) *RingBuffer {
	return &RingBuffer{
		data:     make([]float64, capacity),
		capacity: capacity,
	}
}

// Push adds a value to the buffer, overwriting the oldest if at capacity.
func (rb *RingBuffer) Push(value float64) {
	rb.data[rb.head] = value
	rb.head = (rb.head + 1) % rb.capacity
	if rb.size < rb.capacity {
		rb.size++
	}
}

// Values returns all values in chronological order (oldest first).
// Returns nil if the buffer is empty.
func (rb *RingBuffer) Values() []float64 {
	if rb.size == 0 {
		return nil
	}
	result := make([]float64, rb.size)
	start := (rb.head - rb.size + rb.capacity) % rb.capacity
	for i := 0; i < rb.size; i++ {
		result[i] = rb.data[(start+i)%rb.capacity]
	}
	return result
}

// Len returns the current number of elements in the buffer.
func (rb *RingBuffer) Len() int {
	return rb.size
}

// Latest returns the most recently pushed value and true,
// or 0 and false if the buffer is empty.
func (rb *RingBuffer) Latest() (float64, bool) {
	if rb.size == 0 {
		return 0, false
	}
	// head points to next write position, so latest is at head-1
	idx := (rb.head - 1 + rb.capacity) % rb.capacity
	return rb.data[idx], true
}

// Min returns the minimum value in the buffer, or 0 and false if empty.
func (rb *RingBuffer) Min() (float64, bool) {
	if rb.size == 0 {
		return 0, false
	}
	values := rb.Values()
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min, true
}

// Max returns the maximum value in the buffer, or 0 and false if empty.
func (rb *RingBuffer) Max() (float64, bool) {
	if rb.size == 0 {
		return 0, false
	}
	values := rb.Values()
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max, true
}

// Avg returns the average of all values in the buffer, or 0 and false if empty.
func (rb *RingBuffer) Avg() (float64, bool) {
	if rb.size == 0 {
		return 0, false
	}
	values := rb.Values()
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values)), true
}

// Trend returns 1 (up), -1 (down), or 0 (flat) based on recent values.
// Compares the average of the last 3 values to the average of the first 3 values.
func (rb *RingBuffer) Trend() int {
	if rb.size < 2 {
		return 0
	}
	values := rb.Values()
	n := len(values)

	// Use up to 3 values from each end for comparison
	windowSize := 3
	if n < windowSize*2 {
		windowSize = n / 2
	}
	if windowSize < 1 {
		windowSize = 1
	}

	// Average of first windowSize values
	firstSum := 0.0
	for i := 0; i < windowSize; i++ {
		firstSum += values[i]
	}
	firstAvg := firstSum / float64(windowSize)

	// Average of last windowSize values
	lastSum := 0.0
	for i := n - windowSize; i < n; i++ {
		lastSum += values[i]
	}
	lastAvg := lastSum / float64(windowSize)

	// Threshold for considering values "different" (5% of the first average, minimum 0.01)
	threshold := firstAvg * 0.05
	if threshold < 0.01 {
		threshold = 0.01
	}

	diff := lastAvg - firstAvg
	if diff > threshold {
		return 1 // trending up
	} else if diff < -threshold {
		return -1 // trending down
	}
	return 0 // flat
}

// StdDev returns the standard deviation of values in the buffer.
func (rb *RingBuffer) StdDev() (float64, bool) {
	if rb.size == 0 {
		return 0, false
	}
	avg, _ := rb.Avg()
	values := rb.Values()
	sumSquares := 0.0
	for _, v := range values {
		diff := v - avg
		sumSquares += diff * diff
	}
	return math.Sqrt(sumSquares / float64(len(values))), true
}

// Percentile returns the value at the given percentile (0-100).
// Uses linear interpolation between adjacent values.
func (rb *RingBuffer) Percentile(p float64) (float64, bool) {
	if rb.size == 0 || p < 0 || p > 100 {
		return 0, false
	}
	values := rb.Values()
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	idx := (p / 100) * float64(len(sorted)-1)
	lower := int(idx)
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[len(sorted)-1], true
	}
	weight := idx - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight, true
}

// Median returns the median (p50) value in the buffer.
func (rb *RingBuffer) Median() (float64, bool) {
	return rb.Percentile(50)
}

// Rate returns the rate of change per second given the poll interval.
func (rb *RingBuffer) Rate(interval time.Duration) (float64, bool) {
	if rb.size < 2 {
		return 0, false
	}
	values := rb.Values()
	oldest := values[0]
	latest := values[len(values)-1]
	elapsed := interval.Seconds() * float64(len(values)-1)
	if elapsed == 0 {
		return 0, false
	}
	return (latest - oldest) / elapsed, true
}

// CV returns the coefficient of variation (stddev/mean).
func (rb *RingBuffer) CV() (float64, bool) {
	avg, ok := rb.Avg()
	if !ok || avg == 0 {
		return 0, false
	}
	stddev, _ := rb.StdDev()
	return stddev / avg, true
}
