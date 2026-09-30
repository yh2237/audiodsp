package pitch

import (
	"encoding/binary"
	"math"
	"sync"
	"testing"
)

func testWave(rate, milliseconds int, hz float64) []float64 {
	wave := make([]float64, rate*milliseconds/1000)
	for i := range wave {
		wave[i] = .2*math.Sin(2*math.Pi*hz*float64(i)/float64(rate)) + .03
	}
	return wave
}

func TestEstimateMedianSine(t *testing.T) {
	if got := EstimateMedian(testWave(16000, 500, 220), 16000); math.Abs(got-220) > 3 {
		t.Fatalf("pitch=%g", got)
	}
}

func TestDetectorReuseAndInputOwnership(t *testing.T) {
	var detector Detector
	for _, rate := range []int{48000, 8000, 16000, 8000} {
		for _, milliseconds := range []int{180, 40, 500, 20} {
			wave := testWave(rate, milliseconds, 220)
			original := append([]float64(nil), wave...)
			if hz := detector.EstimateMedian(wave, rate); math.Abs(hz-220) > 3 {
				t.Fatalf("rate=%d length=%d pitch=%g", rate, milliseconds, hz)
			}
			for i := range wave {
				if wave[i] != original[i] {
					t.Fatal("input modified")
				}
			}
		}
	}
	for _, rate := range []int{0, -16000, 1} {
		if detector.EstimateMedian([]float64{.2, -.2}, rate) != 0 {
			t.Fatal("invalid rate/short input voiced")
		}
	}
	for _, level := range []float64{0, .3} {
		wave := make([]float64, 8000)
		for i := range wave {
			wave[i] = level
		}
		if detector.EstimateMedian(wave, 16000) != 0 {
			t.Fatal("silence/DC retained previous pitch")
		}
	}
}

func TestDetectorReusesAllocatedWorkspace(t *testing.T) {
	wave := testWave(16000, 180, 220)
	var detector Detector
	detector.EstimateMedian(wave, 16000)
	if got := testing.AllocsPerRun(5, func() { detector.EstimateMedian(wave, 16000) }); got != 0 {
		t.Fatalf("warmed detector allocations=%g", got)
	}
}

func TestIndependentDetectorsInParallel(t *testing.T) {
	var workers sync.WaitGroup
	for _, frequency := range []float64{110, 220, 440} {
		workers.Add(1)
		go func(hz float64) {
			defer workers.Done()
			var detector Detector
			wave := testWave(16000, 180, hz)
			for i := 0; i < 3; i++ {
				if got := detector.EstimateMedian(wave, 16000); math.Abs(got-hz) > 3 {
					t.Errorf("pitch=%g want=%g", got, hz)
				}
			}
		}(frequency)
	}
	workers.Wait()
}

func FuzzDetector(f *testing.F) {
	f.Add([]byte{0, 0, 1, 0}, 16000)
	f.Add(make([]byte, 640), 8000)
	f.Add([]byte{255, 127, 0, 128}, 0)
	f.Fuzz(func(t *testing.T, data []byte, rate int) {
		if len(data) > 2048 {
			t.Skip()
		}
		wave := make([]float64, len(data)/2)
		for i := range wave {
			wave[i] = float64(int16(binary.LittleEndian.Uint16(data[i*2:]))) / 32768
		}
		var detector Detector
		for _, got := range []float64{detector.Estimate(wave, rate), detector.EstimateMedian(wave, rate)} {
			if got < 0 || math.IsNaN(got) || math.IsInf(got, 0) {
				t.Fatalf("invalid frequency=%g", got)
			}
		}
	})
}
