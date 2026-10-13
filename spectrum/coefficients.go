package spectrum

import (
	"container/list"
	"math"
	"sync"
)

const maxCoefficientBytes = 8 << 20
const maxCoefficientEntries = 32

type coefficientKey struct {
	length, rate, bands int
	low, high           uint64
}

type coefficients struct {
	key                  coefficientKey
	window, cosine, sine []float64
}

func (c *coefficients) bytes() int {
	return (len(c.window) + len(c.cosine) + len(c.sine)) * 8
}

var coefficientCache = struct {
	sync.Mutex
	entries map[coefficientKey]*list.Element
	order   *list.List
	bytes   int
}{entries: make(map[coefficientKey]*list.Element), order: list.New()}

func logCoefficients(key coefficientKey) *coefficients {
	// 大きな窓は保存せず、直接計算する。
	if key.length > maxCoefficientBytes/8 || key.bands > (maxCoefficientBytes/8-key.length)/(2*key.length) {
		return nil
	}
	coefficientCache.Lock()
	if entry := coefficientCache.entries[key]; entry != nil {
		coefficientCache.order.MoveToFront(entry)
		value := entry.Value.(*coefficients)
		coefficientCache.Unlock()
		return value
	}
	coefficientCache.Unlock()
	value := buildCoefficients(key)
	coefficientCache.Lock()
	defer coefficientCache.Unlock()
	if entry := coefficientCache.entries[key]; entry != nil {
		coefficientCache.order.MoveToFront(entry)
		return entry.Value.(*coefficients)
	}
	coefficientCache.entries[key] = coefficientCache.order.PushFront(value)
	coefficientCache.bytes += value.bytes()
	for coefficientCache.bytes > maxCoefficientBytes || len(coefficientCache.entries) > maxCoefficientEntries {
		last := coefficientCache.order.Back()
		old := last.Value.(*coefficients)
		delete(coefficientCache.entries, old.key)
		coefficientCache.bytes -= old.bytes()
		coefficientCache.order.Remove(last)
	}
	return value
}

func buildCoefficients(key coefficientKey) *coefficients {
	c := &coefficients{key: key, window: make([]float64, key.length), cosine: make([]float64, key.length*key.bands), sine: make([]float64, key.length*key.bands)}
	for i := range c.window {
		c.window[i] = .5 - .5*math.Cos(2*math.Pi*float64(i)/float64(key.length-1))
	}
	low, high := math.Float64frombits(key.low), math.Float64frombits(key.high)
	for band := 0; band < key.bands; band++ {
		frequency := low * math.Pow(high/low, float64(band)/float64(key.bands-1))
		for i := 0; i < key.length; i++ {
			angle := 2 * math.Pi * frequency * float64(i) / float64(key.rate)
			c.cosine[band*key.length+i] = math.Cos(angle)
			c.sine[band*key.length+i] = math.Sin(angle)
		}
	}
	return c
}
