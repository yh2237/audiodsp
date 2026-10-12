// correlationパッケージはモノラル波形の平均を除去し、正規化相関を計算する。
package correlation

import "math"

// Normalizedは同じ長さの波形の相関係数を返す。逆相関は負のまま返す。
// 2サンプル未満、長さの不一致、不正なminimumEnergy、低エネルギーでは0を返す。
func Normalized(left, right []float64, minimumEnergy float64) float64 {
	if len(left) < 2 || len(left) != len(right) || !validEnergy(minimumEnergy) {
		return 0
	}
	return normalized(left, right, minimumEnergy)
}

// Maxは-maxLagからmaxLagまで重ねる位置をずらし、最大の正の相関係数を返す。
// minOverlapは各位置で必要なサンプル数で、2以上を指定する。正の相関がなければ0を返す。
func Max(left, right []float64, maxLag, minOverlap int, minimumEnergy float64) float64 {
	if maxLag < 0 || minOverlap < 2 || len(left) < minOverlap || len(right) < minOverlap || !validEnergy(minimumEnergy) {
		return 0
	}
	low := -min(maxLag, len(left)-minOverlap)
	high := min(maxLag, len(right)-minOverlap)
	best := 0.0
	for lag := low; lag <= high; lag++ {
		leftStart, rightStart := max(0, -lag), max(0, lag)
		length := min(len(left)-leftStart, len(right)-rightStart)
		value := normalized(left[leftStart:leftStart+length], right[rightStart:rightStart+length], minimumEnergy)
		best = max(best, value)
	}
	return best
}

func validEnergy(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func normalized(left, right []float64, minimumEnergy float64) float64 {
	leftMean, rightMean := 0.0, 0.0
	for index := range left {
		leftMean += left[index]
		rightMean += right[index]
	}
	leftMean /= float64(len(left))
	rightMean /= float64(len(right))
	cross, leftEnergy, rightEnergy := 0.0, 0.0, 0.0
	for index := range left {
		a := left[index] - leftMean
		b := right[index] - rightMean
		cross += a * b
		leftEnergy += a * a
		rightEnergy += b * b
	}
	if leftEnergy <= minimumEnergy || rightEnergy <= minimumEnergy {
		return 0
	}
	return cross / math.Sqrt(leftEnergy*rightEnergy)
}
