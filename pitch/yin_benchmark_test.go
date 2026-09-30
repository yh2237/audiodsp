package pitch

import "testing"

var benchmarkHz float64

func BenchmarkMedian(b *testing.B) {
	wave := testWave(16000, 180, 220)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkHz = EstimateMedian(wave, 16000)
	}
}

func BenchmarkDetectorMedian(b *testing.B) {
	wave := testWave(16000, 180, 220)
	var detector Detector
	detector.EstimateMedian(wave, 16000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkHz = detector.EstimateMedian(wave, 16000)
	}
}
