package wav

import (
	"bytes"
	"slices"
	"testing"
)

func FuzzDecode(f *testing.F) {
	data, _ := Bytes(signedPCM(2, 8))
	f.Add(data)
	f.Add(riff(chunk("fmt ", fmtBody(1, 1, 8000, 16, 3), true), chunk("JUNK", []byte{1}, true), chunk("data", samplesBody(7), true)))
	f.Add([]byte("RIFF"))
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := Decode(bytes.NewReader(data))
		if err != nil {
			return
		}
		if got.SampleRate <= 0 || got.Channels <= 0 || len(got.Data)%got.Channels != 0 {
			t.Fatalf("invalid PCM accepted: %d Hz, %d channels, %d samples", got.SampleRate, got.Channels, len(got.Data))
		}
		encoded, err := Bytes(got)
		if err != nil {
			// Decodable headers may describe rates or channel counts that the writer rejects.
			if got.Channels > 32767 || uint64(got.SampleRate)*uint64(got.Channels)*2 > 1<<32-1 {
				return
			}
			t.Fatal(err)
		}
		again, err := Decode(bytes.NewReader(encoded))
		if err != nil || again.SampleRate != got.SampleRate || again.Channels != got.Channels ||
			!slices.Equal(again.Data, got.Data) {
			t.Fatalf("re-encoded PCM changed: %v", err)
		}
	})
}
