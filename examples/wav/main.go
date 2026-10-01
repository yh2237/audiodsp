package main

import (
	"bytes"
	"fmt"
	"math"

	"github.com/yh2237/audiodsp/wav"
)

func main() {
	pcm := &wav.PCM{SampleRate: 16000, Channels: 1, Data: make([]int16, 160)}
	for i := range pcm.Data {
		pcm.Data[i] = int16(8000 * math.Sin(2*math.Pi*440*float64(i)/16000))
	}
	var file bytes.Buffer
	if err := wav.Encode(&file, pcm); err != nil {
		panic(err)
	}
	decoded, err := wav.Decode(&file)
	if err != nil {
		panic(err)
	}
	fmt.Println(decoded.SampleRate, decoded.Channels, len(decoded.Data), decoded.Data[:4])
}
