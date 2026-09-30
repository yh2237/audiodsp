// stretchパッケージはモノラル波形の時間伸縮を行う。
package stretch

import "math"

func WSOLA(source []float64, targetFrames, sampleRate int) []float64 {
	ratio := float64(len(source)) / float64(targetFrames)
	return mapped(source, targetFrames, sampleRate, func(output int) float64 {
		return float64(output) * ratio
	})
}

// Anchoredは区間を分割せず、位置配列から作る時間写像に沿って伸縮する。
func Anchored(source []float64, targetFrames, sampleRate int, sourceAnchors, targetAnchors []int) []float64 {
	if len(sourceAnchors) < 2 || len(sourceAnchors) != len(targetAnchors) {
		return WSOLA(source, targetFrames, sampleRate)
	}
	if sourceAnchors[0] < 0 || sourceAnchors[len(sourceAnchors)-1] > len(source) ||
		targetAnchors[0] < 0 || targetAnchors[len(targetAnchors)-1] > targetFrames {
		return WSOLA(source, targetFrames, sampleRate)
	}
	for i := 1; i < len(sourceAnchors); i++ {
		if sourceAnchors[i] <= sourceAnchors[i-1] || targetAnchors[i] <= targetAnchors[i-1] {
			return WSOLA(source, targetFrames, sampleRate)
		}
	}
	segment := 0
	return mapped(source, targetFrames, sampleRate, func(output int) float64 {
		for segment+1 < len(targetAnchors)-1 && output > targetAnchors[segment+1] {
			segment++
		}
		leftTarget, rightTarget := targetAnchors[segment], targetAnchors[segment+1]
		leftSource, rightSource := sourceAnchors[segment], sourceAnchors[segment+1]
		progress := float64(output-leftTarget) / float64(max(1, rightTarget-leftTarget))
		return float64(leftSource) + progress*float64(rightSource-leftSource)
	})
}

func mapped(source []float64, targetFrames, sampleRate int, sourcePosition func(int) float64) []float64 {
	if targetFrames <= 0 || len(source) == 0 {
		return nil
	}
	if len(source) < 16 || targetFrames < 16 {
		return Linear(source, targetFrames)
	}
	window := min(msToFrames(40, sampleRate), len(source), targetFrames)
	if window < 16 {
		return Linear(source, targetFrames)
	}
	if window%2 == 1 {
		window--
	}
	synthesisHop := max(1, window/2)
	search := max(1, min(msToFrames(5, sampleRate), window/4))
	accumulator := make([]float64, targetFrames+window)
	weights := make([]float64, len(accumulator))
	result := make([]float64, targetFrames)
	// 正規化まで出力配列の先頭を窓係数として使い、追加の配列を作らない。
	windowWeights := result[:window]
	for i := range windowWeights {
		windowWeights[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i+1)/float64(window+1))
	}
	previousSource := 0
	maxStart := max(0, len(source)-window)
	for outputPosition := 0; outputPosition < targetFrames; outputPosition += synthesisHop {
		expected := max(0, min(int(math.Round(sourcePosition(outputPosition))), maxStart))
		start := expected
		if outputPosition > 0 {
			start = bestMatch(source, previousSource+synthesisHop, expected, search, window/2, maxStart)
		}
		for i := 0; i < window && outputPosition+i < len(accumulator); i++ {
			weight := windowWeights[i]
			accumulator[outputPosition+i] += source[start+i] * weight
			weights[outputPosition+i] += weight
		}
		previousSource = start
	}
	for i := range result {
		result[i] = 0
		if weights[i] > 1e-12 {
			result[i] = accumulator[i] / weights[i]
		}
	}
	return result
}

func bestMatch(source []float64, reference, expected, search, compare, maxStart int) int {
	reference = max(0, min(reference, maxStart))
	low, high := max(0, expected-search), min(maxStart, expected+search)
	best, bestScore := expected, math.Inf(-1)
	length := min(compare, len(source)-reference, len(source)-high)
	if length < 4 {
		return best
	}
	referenceWave := source[reference : reference+length]
	leftEnergy := 0.0
	for _, value := range referenceWave {
		leftEnergy += value * value
	}
	for candidate := low; candidate <= high; candidate++ {
		numerator, rightEnergy := 0.0, 0.0
		candidateWave := source[candidate : candidate+length]
		for i, left := range referenceWave {
			right := candidateWave[i]
			numerator += left * right
			rightEnergy += right * right
		}
		score := numerator / (math.Sqrt(leftEnergy*rightEnergy) + 1e-12)
		if score > bestScore {
			bestScore, best = score, candidate
		}
	}
	return best
}

// Linearは出力フレーム数に合わせて両端を含む線形補間を行う。
func Linear(source []float64, targetFrames int) []float64 {
	if targetFrames <= 0 || len(source) == 0 {
		return nil
	}
	result := make([]float64, targetFrames)
	if len(source) == 1 {
		for i := range result {
			result[i] = source[0]
		}
		return result
	}
	for i := range result {
		position := float64(i) * float64(len(source)-1) / float64(max(1, targetFrames-1))
		left := int(math.Floor(position))
		right := min(left+1, len(source)-1)
		fraction := position - float64(left)
		result[i] = source[left]*(1-fraction) + source[right]*fraction
	}
	return result
}

func msToFrames(ms float64, sampleRate int) int {
	if ms <= 0 || sampleRate <= 0 {
		return 0
	}
	return int(math.Round(ms * float64(sampleRate) / 1000))
}
