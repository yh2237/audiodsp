package correlation

import (
	"math"
	"testing"
)

func TestNormalized(t *testing.T) {
	left := []float64{0, 1, 2, 3}
	for _, test := range []struct {
		name  string
		right []float64
		want  float64
	}{
		{"same", left, 1},
		{"gain and DC", []float64{7, 9, 11, 13}, 1},
		{"inverted", []float64{6, 4, 2, 0}, -1},
		{"uncorrelated", []float64{1, -1, -1, 1}, 0},
		{"constant", []float64{3, 3, 3, 3}, 0},
		{"length mismatch", []float64{0, 1}, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := Normalized(left, test.right, 1e-12); math.Abs(got-test.want) > 1e-14 {
				t.Fatalf("correlation=%v want=%v", got, test.want)
			}
		})
	}
	if got := Normalized([]float64{0, 2}, []float64{0, 2}, 2); got != 0 {
		t.Fatalf("energy equal to the threshold=%v", got)
	}
	if got := Normalized([]float64{0, 2}, []float64{0, 2}, 1); got != 1 {
		t.Fatalf("energy above the threshold=%v", got)
	}
	for _, threshold := range []float64{-1, math.NaN(), math.Inf(1)} {
		if got := Normalized(left, left, threshold); got != 0 {
			t.Fatalf("invalid threshold=%v correlation=%v", threshold, got)
		}
	}
	if Normalized(nil, nil, 0) != 0 || Normalized([]float64{1}, []float64{1}, 0) != 0 {
		t.Fatal("short input was accepted")
	}
}

func TestMaxUsesLagAndMinimumOverlap(t *testing.T) {
	left := []float64{1, 2, 3, 4}
	right := []float64{9, 1, 2, 3, 4}
	if got := Max(left, right, 0, 4, 0); got != 0 {
		t.Fatalf("zero-lag positive correlation=%v", got)
	}
	for _, waves := range [][2][]float64{{left, right}, {right, left}} {
		if got := Max(waves[0], waves[1], 1, 4, 0); math.Abs(got-1) > 1e-14 {
			t.Fatalf("lag correlation=%v", got)
		}
		if got := Max(waves[0], waves[1], int(^uint(0)>>1), 4, 0); math.Abs(got-1) > 1e-14 {
			t.Fatalf("large lag correlation=%v", got)
		}
	}
	if Max(left, right, 1, 5, 0) != 0 || Max(left, right, -1, 4, 0) != 0 || Max(left, right, 1, 1, 0) != 0 {
		t.Fatal("invalid lag or overlap was accepted")
	}
	if got := Max(left, []float64{4, 3, 2, 1}, 0, 4, 0); got != 0 {
		t.Fatalf("negative correlation=%v", got)
	}
	if got := Max(left, right, 1, 4, 100); got != 0 {
		t.Fatalf("low-energy correlation=%v", got)
	}
}

func FuzzCorrelation(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 7, 9, 11, 13}, uint8(0), uint8(2))
	f.Add([]byte{0, 0}, uint8(5), uint8(2))
	f.Fuzz(func(t *testing.T, data []byte, lag, overlap uint8) {
		if len(data) > 512 {
			data = data[:512]
		}
		left, right := make([]float64, len(data)/2), make([]float64, len(data)/2)
		for i := range left {
			left[i] = float64(int8(data[i])) / 128
			right[i] = float64(int8(data[i+len(left)])) / 128
		}
		for _, result := range []float64{Normalized(left, right, 1e-12), Max(left, right, int(lag), max(2, int(overlap)), 1e-10)} {
			if math.IsNaN(result) || math.IsInf(result, 0) || math.Abs(result) > 1+1e-12 {
				t.Fatalf("correlation=%v", result)
			}
		}
	})
}
