package spectrum

import "testing"

var benchmarkSpectrum []float64

func BenchmarkLog(b *testing.B) {
	wave := testWave(640)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkSpectrum = Log(wave, 16000, 24, 100, 7200)
	}
}
