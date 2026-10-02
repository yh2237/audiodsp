// pitchパッケージはモノラル波形から周波数を推定する。
package pitch

import (
	"math"
	"slices"
)

// Detectorは作業配列を再利用する。ゼロ値で使えるが、同じ値を並行して呼ばない。
type Detector struct {
	difference []float64
	values     []float64
}

func EstimateMedian(wave []float64, sampleRate int) float64 {
	var detector Detector
	return detector.EstimateMedian(wave, sampleRate)
}

func (d *Detector) EstimateMedian(wave []float64, sampleRate int) float64 {
	if sampleRate <= 0 {
		return 0
	}
	window := frames(40, sampleRate)
	hop := max(1, frames(10, sampleRate))
	if window < 1 {
		return 0
	}
	if len(wave) < window {
		return d.Estimate(wave, sampleRate)
	}
	count := (len(wave)-window)/hop + 1
	if cap(d.values) < count {
		d.values = make([]float64, 0, count)
	}
	values := d.values[:0]
	for start := 0; start+window <= len(wave); start += hop {
		if value := d.Estimate(wave[start:start+window], sampleRate); value > 0 {
			values = append(values, value)
		}
	}
	d.values = values
	if len(values) == 0 {
		return 0
	}
	slices.Sort(values)
	middle := len(values) / 2
	if len(values)%2 == 1 {
		return values[middle]
	}
	return (values[middle-1] + values[middle]) / 2
}

func Estimate(frame []float64, sampleRate int) float64 {
	var detector Detector
	return detector.Estimate(frame, sampleRate)
}

func (d *Detector) Estimate(frame []float64, sampleRate int) float64 {
	if sampleRate <= 0 || len(frame) < 16 || rms(frame) < 0.003 {
		return 0
	}
	mean := 0.0
	for _, value := range frame {
		mean += value
	}
	mean /= float64(len(frame))
	centeredEnergy := 0.0
	for _, value := range frame {
		delta := value - mean
		centeredEnergy += delta * delta
	}
	if math.Sqrt(centeredEnergy/float64(len(frame))) < 0.003 {
		return 0
	}
	minLag, maxLag := max(2, sampleRate/500), min(len(frame)/2, sampleRate/60)
	if minLag >= maxLag {
		return 0
	}
	if cap(d.difference) < maxLag+1 {
		d.difference = make([]float64, maxLag+1)
	}
	difference := d.difference[:maxLag+1]
	for lag := 1; lag <= maxLag; lag++ {
		sum := 0.0
		left, right := frame[:len(frame)-lag], frame[lag:]
		for i, value := range left {
			delta := (value - mean) - (right[i] - mean)
			sum += delta * delta
		}
		difference[lag] = sum
	}
	cumulative := 0.0
	for lag := 1; lag <= maxLag; lag++ {
		cumulative += difference[lag]
		if lag >= minLag && cumulative > 0 {
			difference[lag] = difference[lag] * float64(lag) / cumulative
		}
	}
	bestValue, bestLag := math.Inf(1), 0
	for lag := minLag; lag <= maxLag; lag++ {
		if difference[lag] < bestValue {
			bestValue, bestLag = difference[lag], lag
		}
		if difference[lag] < 0.15 && (lag == maxLag || difference[lag+1] >= difference[lag]) {
			bestLag, bestValue = lag, difference[lag]
			break
		}
	}
	if bestLag == 0 {
		return 0
	}
	bestLag, bestValue = fundamentalLag(difference, bestLag, bestValue, minLag)
	if bestValue > 0.35 {
		return 0
	}
	refined := float64(bestLag)
	if bestLag > minLag && bestLag < maxLag {
		left, center, right := difference[bestLag-1], difference[bestLag], difference[bestLag+1]
		denominator := left - 2*center + right
		if math.Abs(denominator) > 1e-12 {
			refined += 0.5 * (left - right) / denominator
		}
	}
	return float64(sampleRate) / refined
}

// fundamentalLagは、周期の整数倍（分周）を選んだ場合に、近い深さの谷がある1/2〜1/4の遅れを基本周期として選び直す。
// 声門の揺れなどで数周期ごとの繰り返しのほうが似ている声では、1周期の谷がわずかに浅くなる。
// 深さの許容は選んだ谷に比例させ、深い谷（ほぼ完全な周期）では倍音を基本周期と取り違えないようにする。
func fundamentalLag(difference []float64, lag int, value float64, minLag int) (int, float64) {
	for divisor := 4; divisor >= 2; divisor-- {
		center := float64(lag) / float64(divisor)
		low := max(minLag, int(math.Floor(center*0.96)))
		high := min(len(difference)-2, int(math.Ceil(center*1.04)))
		candidate, candidateValue := 0, math.Inf(1)
		for index := max(low, 1); index <= high; index++ {
			if difference[index] < candidateValue && difference[index] <= difference[index-1] && difference[index] <= difference[index+1] {
				candidate, candidateValue = index, difference[index]
			}
		}
		if candidate > 0 && candidateValue <= value*1.5+0.1 {
			return candidate, candidateValue
		}
	}
	return lag, value
}

func rms(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, value := range values {
		sum += value * value
	}
	return math.Sqrt(sum / float64(len(values)))
}

func frames(ms float64, sampleRate int) int { return int(math.Round(ms * float64(sampleRate) / 1000)) }
