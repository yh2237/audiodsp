package main

import (
	"fmt"
	"math"

	"github.com/yh2237/audiodsp/spectrum"
)

func main() {
	wave := make([]float64, 640)
	for i := range wave {
		wave[i] = .25 * math.Sin(2*math.Pi*1000*float64(i)/16000)
	}
	fmt.Println(spectrum.Log(wave, 16000, 3, 500, 2000))
}
