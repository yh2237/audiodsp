package main

import (
	"fmt"
	"math"

	"github.com/yh2237/audiodsp/pitch"
)

func main() {
	const rate = 16000
	wave := make([]float64, rate/2)
	for i := range wave {
		wave[i] = .2 * math.Sin(2*math.Pi*220*float64(i)/rate)
	}
	var detector pitch.Detector
	fmt.Printf("%.2f Hz\n", detector.EstimateMedian(wave, rate))
}
