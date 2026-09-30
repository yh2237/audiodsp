package stretch

import "testing"

var benchmarkWave []float64

func BenchmarkWSOLA(b *testing.B) {
	source := testWave()
	for _, tc := range []struct {
		name   string
		frames int
	}{{"compress", 3200}, {"expand", 7200}} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchmarkWave = WSOLA(source, tc.frames, 16000)
			}
		})
	}
}
