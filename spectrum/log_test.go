package spectrum

import (
	"math"
	"testing"
)

func testWave(length int) []float64 {
	wave := make([]float64, length)
	for i := range wave {
		wave[i] = .25 * math.Sin(2*math.Pi*1000*float64(i)/16000)
	}
	return wave
}

func TestLogFindsKnownFrequencyAndPreservesInput(t *testing.T) {
	for _, length := range []int{640, 5000} {
		wave := testWave(length)
		original := append([]float64(nil), wave...)
		values := Log(wave, 16000, 3, 500, 2000)
		if len(values) != 3 || values[1] <= values[0]+20 || values[1] <= values[2]+20 {
			t.Fatal("known frequency did not dominate", values)
		}
		for i := range wave {
			if wave[i] != original[i] {
				t.Fatal("input modified")
			}
		}
		values[0] = 123
		if Log(wave, 16000, 3, 500, 2000)[0] == 123 {
			t.Fatal("outputs share storage")
		}
	}
}

func TestLogSilenceFloorAndInvalidConfiguration(t *testing.T) {
	for _, value := range Log(make([]float64, 640), 16000, 24, 100, 7200) {
		if value != -140 {
			t.Fatalf("silence=%g", value)
		}
	}
	for _, tc := range []struct {
		rate, bands int
		low, high   float64
	}{
		{0, 24, 100, 7200}, {16000, 1, 100, 7200}, {16000, 24, 0, 7200}, {16000, 24, 100, 100},
		{16000, 24, math.NaN(), 7200}, {16000, 24, 100, math.Inf(1)},
	} {
		if Log(testWave(640), tc.rate, tc.bands, tc.low, tc.high) != nil {
			t.Fatal("invalid configuration accepted")
		}
	}
	if Log([]float64{1}, 16000, 24, 100, 7200) != nil {
		t.Fatal("single sample accepted")
	}
}

func FuzzLog(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3}, 16000, 4)
	f.Add(make([]byte, 64), 8000, 10)
	f.Fuzz(func(t *testing.T, data []byte, rate, bands int) {
		if len(data) > 1024 || rate < 1 || rate > 48000 || bands < 2 || bands > 32 {
			t.Skip()
		}
		wave := make([]float64, len(data))
		for i, v := range data {
			wave[i] = float64(int8(v)) / 128
		}
		values := Log(wave, rate, bands, 10, float64(rate)/2)
		for _, v := range values {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Fatal("non-finite output")
			}
		}
	})
}
