package stretch

import (
	"encoding/binary"
	"math"
	"reflect"
	"testing"
)

func testWave() []float64 {
	wave := make([]float64, 4800)
	for i := range wave {
		wave[i] = .25 * math.Sin(2*math.Pi*220*float64(i)/16000)
	}
	return wave
}

func TestWSOLALengthPitchAndOwnership(t *testing.T) {
	source := testWave()
	original := append([]float64(nil), source...)
	for _, target := range []int{3200, 7200} {
		first := WSOLA(source, target, 16000)
		if len(first) != target || !reflect.DeepEqual(first, WSOLA(source, target, 16000)) {
			t.Fatal("length or determinism changed")
		}
		crossings := 0
		for i := 1; i < len(first); i++ {
			if first[i-1] <= 0 && first[i] > 0 {
				crossings++
			}
		}
		if hz := float64(crossings) * 16000 / float64(target); math.Abs(hz-220) > 15 {
			t.Fatalf("pitch changed: %g", hz)
		}
		first[0] = 42
		if WSOLA(source, target, 16000)[0] == 42 {
			t.Fatal("outputs share storage")
		}
	}
	if !reflect.DeepEqual(source, original) {
		t.Fatal("input modified")
	}
}

func TestAnchoredAndInvalidPositions(t *testing.T) {
	source := testWave()
	anchors := []int{0, 1600, 3200, 4800}
	target := []int{0, 800, 2400, 4000}
	want := Anchored(source, 4000, 16000, anchors, target)
	if len(want) != 4000 || !reflect.DeepEqual(want, Anchored(source, 4000, 16000, anchors, target)) {
		t.Fatal("anchored output invalid")
	}
	for _, bad := range [][]int{{0}, {0, 0, 3200, 4800}, {-1, 1600, 3200, 4800}, {0, 1600, 3200, 5000}} {
		if !reflect.DeepEqual(Anchored(source, 4000, 16000, bad, target), WSOLA(source, 4000, 16000)) {
			t.Fatal("invalid anchors did not use uniform stretch")
		}
	}
	if len(Anchored(source, 4000, 16000, []int{0, 4800}, []int{1000, 4000})) != 4000 {
		t.Fatal("partial anchors failed")
	}
}

func TestLinearFallbackAndEdges(t *testing.T) {
	if got := Linear([]float64{0, 1}, 3); !reflect.DeepEqual(got, []float64{0, .5, 1}) {
		t.Fatal(got)
	}
	if got := WSOLA([]float64{.2}, 3, 16000); !reflect.DeepEqual(got, []float64{.2, .2, .2}) {
		t.Fatal(got)
	}
	if WSOLA(nil, 30, 16000) != nil || WSOLA(testWave(), 0, 16000) != nil {
		t.Fatal("empty input/output changed")
	}
	if !reflect.DeepEqual(WSOLA(testWave(), 32, 0), Linear(testWave(), 32)) {
		t.Fatal("nonpositive rate fallback changed")
	}
}

func FuzzStretch(f *testing.F) {
	f.Add([]byte{0, 0, 1, 0}, 32, 16000, 0)
	f.Add(make([]byte, 128), 80, 8000, 4)
	f.Fuzz(func(t *testing.T, data []byte, target, rate, firstTarget int) {
		if len(data) > 512 || target < 0 || target > 512 || rate < -1 || rate > 48000 {
			t.Skip()
		}
		wave := make([]float64, len(data)/2)
		for i := range wave {
			wave[i] = float64(int16(binary.LittleEndian.Uint16(data[i*2:]))) / 32768
		}
		outputs := [][]float64{WSOLA(wave, target, rate), Anchored(wave, target, rate, []int{0, len(wave)}, []int{firstTarget, target})}
		for _, output := range outputs {
			if len(wave) > 0 && target > 0 && len(output) != target {
				t.Fatal("incorrect output length")
			}
			for _, v := range output {
				if math.IsNaN(v) || math.IsInf(v, 0) {
					t.Fatal("non-finite output")
				}
			}
		}
	})
}
