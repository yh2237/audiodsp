package spectrum

import (
	"math"
	"reflect"
	"sync"
	"testing"
)

func directLog(wave []float64, rate, bands int, low, high float64) []float64 {
	weighted := make([]float64, len(wave))
	for i, v := range wave {
		weighted[i] = v * (.5 - .5*math.Cos(2*math.Pi*float64(i)/float64(len(wave)-1)))
	}
	out := make([]float64, bands)
	for band := range out {
		frequency := low * math.Pow(high/low, float64(band)/float64(bands-1))
		var real, imaginary float64
		for i, value := range weighted {
			angle := 2 * math.Pi * frequency * float64(i) / float64(rate)
			real += value * math.Cos(angle)
			imaginary -= value * math.Sin(angle)
		}
		out[band] = 20 * math.Log10(max(math.Hypot(real, imaginary)/float64(len(wave)), 1e-7))
	}
	return out
}

func TestCachedCoefficientsKeepDirectDFTResults(t *testing.T) {
	for _, length := range []int{32, 640, 1322, 5000} {
		for _, rate := range []int{8000, 16000, 44100} {
			wave := testWave(length)
			got := Log(wave, rate, 10, 100, float64(rate)*.45)
			want := directLog(wave, rate, 10, 100, float64(rate)*.45)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("DFT changed: length=%d rate=%d", length, rate)
			}
		}
	}
}

func TestCoefficientCacheLimitsAndConcurrency(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			wave := testWave(640 + i)
			want := directLog(wave, 16000, 24, 100, 7200)
			if !reflect.DeepEqual(Log(wave, 16000, 24, 100, 7200), want) {
				t.Error("concurrent output changed")
			}
		}(i)
	}
	wg.Wait()
	coefficientCache.Lock()
	if coefficientCache.bytes > maxCoefficientBytes || len(coefficientCache.entries) > maxCoefficientEntries {
		t.Error("cache exceeds limits")
	}
	coefficientCache.Unlock()
	if logCoefficients(coefficientKey{length: maxCoefficientBytes, bands: 24}) != nil {
		t.Fatal("oversized kernel retained")
	}
	wave := testWave(6000)
	if !reflect.DeepEqual(Log(wave, 16000, 100, 100, 7200), directLog(wave, 16000, 100, 100, 7200)) {
		t.Fatal("uncached output changed")
	}
}

func BenchmarkLogDirectReference(b *testing.B) {
	wave := testWave(640)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkSpectrum = directLog(wave, 16000, 24, 100, 7200)
	}
}
