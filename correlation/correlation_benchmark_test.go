package correlation

import (
	"math"
	"testing"
)

var benchmarkCorrelation float64

func BenchmarkMax(b *testing.B) {
	left, right := make([]float64, 240), make([]float64, 240)
	for i := range left {
		left[i] = math.Sin(float64(i) * 0.31)
		right[i] = math.Sin(float64(i-7) * 0.31)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkCorrelation = Max(left, right, 40, 32, 1e-10)
	}
}
