package main

import (
	"fmt"
	"math"

	"github.com/yh2237/audiodsp/stretch"
)

func main() {
	wave := make([]float64, 4800)
	for i := range wave {
		wave[i] = .25 * math.Sin(2*math.Pi*220*float64(i)/16000)
	}
	output := stretch.WSOLA(wave, 7200, 16000)
	fmt.Printf("%d -> %d mono frames\n", len(wave), len(output))
}
