package main

import (
	"fmt"
	"math"

	"github.com/yh2237/audiodsp/correlation"
)

func main() {
	left, right := make([]float64, 64), make([]float64, 64)
	for i := range left {
		left[i] = math.Sin(float64(i)*0.31) + 0.2*math.Cos(float64(i)*0.79)
		if i >= 7 {
			right[i] = 2*left[i-7] + 0.3
		}
	}
	fmt.Printf("zero lag: %.6f\n", correlation.Normalized(left, right, 1e-12))
	fmt.Printf("within 12 samples: %.6f\n", correlation.Max(left, right, 12, 32, 1e-10))
}
