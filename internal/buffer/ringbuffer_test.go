package buffer

import (
	"math"
	"testing"
	"time"
)

func TestRingBuffer_PushSingleValue(t *testing.T) {
	rb := New(30)
	rb.Push(42.0)

	values := rb.Values()
	if len(values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(values))
	}
	if values[0] != 42.0 {
		t.Errorf("expected 42.0, got %f", values[0])
	}
}

func TestRingBuffer_PushFillsToCapacity(t *testing.T) {
	rb := New(30)

	for i := 0; i < 30; i++ {
		rb.Push(float64(i))
	}

	values := rb.Values()
	if len(values) != 30 {
		t.Fatalf("expected 30 values, got %d", len(values))
	}

	// Verify all values in order
	for i := 0; i < 30; i++ {
		if values[i] != float64(i) {
			t.Errorf("at index %d: expected %f, got %f", i, float64(i), values[i])
		}
	}
}

func TestRingBuffer_PushOverflow(t *testing.T) {
	rb := New(30)

	// Push 31 values (0 through 30)
	for i := 0; i <= 30; i++ {
		rb.Push(float64(i))
	}

	values := rb.Values()
	if len(values) != 30 {
		t.Fatalf("expected 30 values, got %d", len(values))
	}

	// First value should be 1 (0 was dropped)
	if values[0] != 1.0 {
		t.Errorf("expected first value 1.0, got %f", values[0])
	}

	// Last value should be 30
	if values[29] != 30.0 {
		t.Errorf("expected last value 30.0, got %f", values[29])
	}
}

func TestRingBuffer_ValuesChronological(t *testing.T) {
	rb := New(30)

	// Push 35 values (0 through 34)
	for i := 0; i < 35; i++ {
		rb.Push(float64(i))
	}

	values := rb.Values()
	if len(values) != 30 {
		t.Fatalf("expected 30 values, got %d", len(values))
	}

	// Should contain values 5-34 in chronological order
	for i := 0; i < 30; i++ {
		expected := float64(i + 5)
		if values[i] != expected {
			t.Errorf("at index %d: expected %f, got %f", i, expected, values[i])
		}
	}
}

func TestRingBuffer_EmptyReturnsNil(t *testing.T) {
	rb := New(30)

	values := rb.Values()
	if values != nil {
		t.Errorf("expected nil for empty buffer, got %v", values)
	}
}

func TestRingBuffer_PushNaN(t *testing.T) {
	rb := New(30)
	rb.Push(math.NaN())

	values := rb.Values()
	if len(values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(values))
	}

	if !math.IsNaN(values[0]) {
		t.Errorf("expected NaN, got %f", values[0])
	}
}

func TestRingBuffer_PushInf(t *testing.T) {
	rb := New(30)
	rb.Push(math.Inf(1))  // +Inf
	rb.Push(math.Inf(-1)) // -Inf

	values := rb.Values()
	if len(values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(values))
	}

	if !math.IsInf(values[0], 1) {
		t.Errorf("expected +Inf, got %f", values[0])
	}
	if !math.IsInf(values[1], -1) {
		t.Errorf("expected -Inf, got %f", values[1])
	}
}

func TestRingBuffer_Len(t *testing.T) {
	rb := New(30)

	if rb.Len() != 0 {
		t.Errorf("expected len 0, got %d", rb.Len())
	}

	rb.Push(1.0)
	if rb.Len() != 1 {
		t.Errorf("expected len 1, got %d", rb.Len())
	}

	// Fill to capacity
	for i := 1; i < 30; i++ {
		rb.Push(float64(i))
	}
	if rb.Len() != 30 {
		t.Errorf("expected len 30, got %d", rb.Len())
	}

	// Overflow should still be 30
	rb.Push(100.0)
	if rb.Len() != 30 {
		t.Errorf("after overflow: expected len 30, got %d", rb.Len())
	}
}

func TestRingBuffer_Latest(t *testing.T) {
	rb := New(30)

	// Empty buffer
	_, ok := rb.Latest()
	if ok {
		t.Error("expected ok=false for empty buffer")
	}

	rb.Push(42.0)
	val, ok := rb.Latest()
	if !ok {
		t.Error("expected ok=true after push")
	}
	if val != 42.0 {
		t.Errorf("expected 42.0, got %f", val)
	}

	rb.Push(99.0)
	val, ok = rb.Latest()
	if !ok {
		t.Error("expected ok=true")
	}
	if val != 99.0 {
		t.Errorf("expected 99.0, got %f", val)
	}
}

func TestRingBuffer_Min(t *testing.T) {
	rb := New(30)

	// Empty buffer
	_, ok := rb.Min()
	if ok {
		t.Error("expected ok=false for empty buffer")
	}

	rb.Push(5.0)
	rb.Push(2.0)
	rb.Push(8.0)
	rb.Push(1.0)
	rb.Push(9.0)

	min, ok := rb.Min()
	if !ok {
		t.Error("expected ok=true")
	}
	if min != 1.0 {
		t.Errorf("expected 1.0, got %f", min)
	}
}

func TestRingBuffer_Max(t *testing.T) {
	rb := New(30)

	// Empty buffer
	_, ok := rb.Max()
	if ok {
		t.Error("expected ok=false for empty buffer")
	}

	rb.Push(5.0)
	rb.Push(2.0)
	rb.Push(8.0)
	rb.Push(1.0)
	rb.Push(9.0)

	max, ok := rb.Max()
	if !ok {
		t.Error("expected ok=true")
	}
	if max != 9.0 {
		t.Errorf("expected 9.0, got %f", max)
	}
}

func TestRingBuffer_Avg(t *testing.T) {
	rb := New(30)

	// Empty buffer
	_, ok := rb.Avg()
	if ok {
		t.Error("expected ok=false for empty buffer")
	}

	rb.Push(10.0)
	rb.Push(20.0)
	rb.Push(30.0)

	avg, ok := rb.Avg()
	if !ok {
		t.Error("expected ok=true")
	}
	if avg != 20.0 {
		t.Errorf("expected 20.0, got %f", avg)
	}
}

func TestRingBuffer_Trend(t *testing.T) {
	rb := New(30)

	// Empty buffer - flat
	if rb.Trend() != 0 {
		t.Error("expected trend=0 for empty buffer")
	}

	// Single value - flat
	rb.Push(5.0)
	if rb.Trend() != 0 {
		t.Error("expected trend=0 for single value")
	}

	// Upward trend
	rb = New(30)
	rb.Push(1.0)
	rb.Push(2.0)
	rb.Push(3.0)
	rb.Push(4.0)
	rb.Push(5.0)
	if rb.Trend() != 1 {
		t.Errorf("expected trend=1 (up), got %d", rb.Trend())
	}

	// Downward trend
	rb = New(30)
	rb.Push(5.0)
	rb.Push(4.0)
	rb.Push(3.0)
	rb.Push(2.0)
	rb.Push(1.0)
	if rb.Trend() != -1 {
		t.Errorf("expected trend=-1 (down), got %d", rb.Trend())
	}

	// Flat trend (same values)
	rb = New(30)
	rb.Push(5.0)
	rb.Push(5.0)
	rb.Push(5.0)
	if rb.Trend() != 0 {
		t.Errorf("expected trend=0 (flat), got %d", rb.Trend())
	}
}

func TestRingBuffer_StdDev(t *testing.T) {
	rb := New(30)

	// Empty buffer
	_, ok := rb.StdDev()
	if ok {
		t.Error("expected ok=false for empty buffer")
	}

	// Single value - stddev should be 0
	rb.Push(5.0)
	stddev, ok := rb.StdDev()
	if !ok {
		t.Error("expected ok=true")
	}
	if stddev != 0.0 {
		t.Errorf("expected 0.0 for single value, got %f", stddev)
	}

	// Known values: [2, 4, 4, 4, 5, 5, 7, 9]
	// Mean = 5, Variance = 4, StdDev = 2
	rb = New(30)
	for _, v := range []float64{2, 4, 4, 4, 5, 5, 7, 9} {
		rb.Push(v)
	}
	stddev, ok = rb.StdDev()
	if !ok {
		t.Error("expected ok=true")
	}
	if math.Abs(stddev-2.0) > 0.001 {
		t.Errorf("expected stddev ~2.0, got %f", stddev)
	}
}

func TestRingBuffer_Percentile(t *testing.T) {
	rb := New(30)

	// Empty buffer
	_, ok := rb.Percentile(50)
	if ok {
		t.Error("expected ok=false for empty buffer")
	}

	// Invalid percentile
	rb.Push(1.0)
	_, ok = rb.Percentile(-1)
	if ok {
		t.Error("expected ok=false for negative percentile")
	}
	_, ok = rb.Percentile(101)
	if ok {
		t.Error("expected ok=false for percentile > 100")
	}

	// Single value - all percentiles return that value
	p50, ok := rb.Percentile(50)
	if !ok || p50 != 1.0 {
		t.Errorf("expected 1.0 for single value, got %f", p50)
	}

	// Values 1-10: p0=1, p50=5.5, p100=10
	rb = New(30)
	for i := 1; i <= 10; i++ {
		rb.Push(float64(i))
	}

	p0, _ := rb.Percentile(0)
	if p0 != 1.0 {
		t.Errorf("expected p0=1.0, got %f", p0)
	}

	p50, _ = rb.Percentile(50)
	if math.Abs(p50-5.5) > 0.001 {
		t.Errorf("expected p50=5.5, got %f", p50)
	}

	p100, _ := rb.Percentile(100)
	if p100 != 10.0 {
		t.Errorf("expected p100=10.0, got %f", p100)
	}

	p95, _ := rb.Percentile(95)
	// 95th percentile of 1-10: idx = 0.95 * 9 = 8.55
	// interpolate between values[8]=9 and values[9]=10
	// 9 * 0.45 + 10 * 0.55 = 9.55
	if math.Abs(p95-9.55) > 0.001 {
		t.Errorf("expected p95=9.55, got %f", p95)
	}
}

func TestRingBuffer_Median(t *testing.T) {
	rb := New(30)

	// Empty buffer
	_, ok := rb.Median()
	if ok {
		t.Error("expected ok=false for empty buffer")
	}

	// Odd number of values: 1, 2, 3, 4, 5
	for i := 1; i <= 5; i++ {
		rb.Push(float64(i))
	}
	median, ok := rb.Median()
	if !ok {
		t.Error("expected ok=true")
	}
	if median != 3.0 {
		t.Errorf("expected median=3.0, got %f", median)
	}

	// Even number of values: 1, 2, 3, 4
	rb = New(30)
	for i := 1; i <= 4; i++ {
		rb.Push(float64(i))
	}
	median, ok = rb.Median()
	if !ok {
		t.Error("expected ok=true")
	}
	if median != 2.5 {
		t.Errorf("expected median=2.5, got %f", median)
	}
}

func TestRingBuffer_Rate(t *testing.T) {
	rb := New(30)
	interval := time.Second

	// Empty buffer
	_, ok := rb.Rate(interval)
	if ok {
		t.Error("expected ok=false for empty buffer")
	}

	// Single value - need at least 2
	rb.Push(10.0)
	_, ok = rb.Rate(interval)
	if ok {
		t.Error("expected ok=false for single value")
	}

	// Two values: 10, 20 with 1 second interval
	// Rate = (20-10) / 1 = 10 per second
	rb.Push(20.0)
	rate, ok := rb.Rate(interval)
	if !ok {
		t.Error("expected ok=true")
	}
	if math.Abs(rate-10.0) > 0.001 {
		t.Errorf("expected rate=10.0, got %f", rate)
	}

	// Five values: 0, 10, 20, 30, 40 with 1 second interval
	// Rate = (40-0) / 4 = 10 per second
	rb = New(30)
	for i := 0; i <= 4; i++ {
		rb.Push(float64(i * 10))
	}
	rate, ok = rb.Rate(interval)
	if !ok {
		t.Error("expected ok=true")
	}
	if math.Abs(rate-10.0) > 0.001 {
		t.Errorf("expected rate=10.0, got %f", rate)
	}

	// Same values with 2 second interval
	// Rate = (40-0) / 8 = 5 per second
	rate, ok = rb.Rate(2 * time.Second)
	if !ok {
		t.Error("expected ok=true")
	}
	if math.Abs(rate-5.0) > 0.001 {
		t.Errorf("expected rate=5.0, got %f", rate)
	}

	// Zero interval should return false
	_, ok = rb.Rate(0)
	if ok {
		t.Error("expected ok=false for zero interval")
	}
}

func TestRingBuffer_CV(t *testing.T) {
	rb := New(30)

	// Empty buffer
	_, ok := rb.CV()
	if ok {
		t.Error("expected ok=false for empty buffer")
	}

	// Single value with mean 0 should return false
	rb.Push(0.0)
	_, ok = rb.CV()
	if ok {
		t.Error("expected ok=false when mean is zero")
	}

	// Known values: [2, 4, 4, 4, 5, 5, 7, 9]
	// Mean = 5, StdDev = 2, CV = 2/5 = 0.4
	rb = New(30)
	for _, v := range []float64{2, 4, 4, 4, 5, 5, 7, 9} {
		rb.Push(v)
	}
	cv, ok := rb.CV()
	if !ok {
		t.Error("expected ok=true")
	}
	if math.Abs(cv-0.4) > 0.001 {
		t.Errorf("expected cv=0.4, got %f", cv)
	}
}
