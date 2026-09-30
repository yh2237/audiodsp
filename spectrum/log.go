// spectrumパッケージは対数間隔の周波数点で窓付き振幅を求める。
package spectrum

import "math"

// LogはHann窓を掛けた直接DFTの振幅を、1e-7下限のdBで返す。入力は変更しない。
func Log(values []float64, sampleRate, bands int, minimumHz, maximumHz float64) []float64 {
	if len(values) < 2 || sampleRate <= 0 || bands < 2 || minimumHz <= 0 || maximumHz <= minimumHz ||
		math.IsNaN(minimumHz) || math.IsNaN(maximumHz) || math.IsInf(minimumHz, 0) || math.IsInf(maximumHz, 0) {
		return nil
	}
	ratio := maximumHz / minimumHz
	if math.IsInf(ratio, 0) || math.IsInf(2*math.Pi*maximumHz*float64(len(values)-1)/float64(sampleRate), 0) {
		return nil
	}
	result := make([]float64, bands)
	var scratch [4096]float64
	weighted := scratch[:min(len(values), len(scratch))]
	if len(values) > len(scratch) {
		weighted = make([]float64, len(values))
	}
	for i, value := range values {
		window := 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(len(values)-1))
		weighted[i] = value * window
	}
	for band := range result {
		frequency := minimumHz * math.Pow(ratio, float64(band)/float64(bands-1))
		var real, imaginary float64
		for i, value := range weighted {
			angle := 2 * math.Pi * frequency * float64(i) / float64(sampleRate)
			real += value * math.Cos(angle)
			imaginary -= value * math.Sin(angle)
		}
		amplitude := math.Hypot(real, imaginary) / float64(len(weighted))
		result[band] = 20 * math.Log10(max(amplitude, 1e-7))
	}
	return result
}
