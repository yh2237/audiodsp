package wav

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"testing"
)

var benchmarkPCM *PCM
var benchmarkBytes []byte

func benchmarkAudio(rate, channels int) *PCM {
	data := make([]int16, rate*channels)
	for frame := 0; frame < rate; frame++ {
		for channel := 0; channel < channels; channel++ {
			data[frame*channels+channel] = int16(12000 * math.Sin(2*math.Pi*220*float64(frame)/float64(rate)))
		}
	}
	return &PCM{SampleRate: rate, Channels: channels, Data: data}
}

func BenchmarkDecode(b *testing.B) {
	for _, channels := range []int{1, 2} {
		b.Run(fmt.Sprintf("48000Hz/%dch", channels), func(b *testing.B) {
			data, _ := Bytes(benchmarkAudio(48000, channels))
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				var err error
				if benchmarkPCM, err = Decode(bytes.NewReader(data)); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkEncode(b *testing.B) {
	pcm := benchmarkAudio(48000, 1)
	b.ReportAllocs()
	b.SetBytes(int64(44 + len(pcm.Data)*2))
	for i := 0; i < b.N; i++ {
		if err := Encode(io.Discard, pcm); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBytes(b *testing.B) {
	pcm := benchmarkAudio(48000, 1)
	b.ReportAllocs()
	b.SetBytes(int64(44 + len(pcm.Data)*2))
	for i := 0; i < b.N; i++ {
		benchmarkBytes, _ = Bytes(pcm)
	}
}
