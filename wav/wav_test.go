package wav

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"reflect"
	"runtime"
	"slices"
	"testing"
)

type fragmentReader struct{ reader io.Reader }

func (r fragmentReader) Read(data []byte) (int, error) {
	return r.reader.Read(data[:min(3, len(data))])
}

func signedPCM(channels, samples int) *PCM {
	pcm := &PCM{SampleRate: 48000, Channels: channels, Data: make([]int16, samples)}
	values := []int16{-32768, -1, 0, 32767}
	for i := range pcm.Data {
		pcm.Data[i] = values[i%len(values)]
	}
	return pcm
}

func mustBytes(t *testing.T, pcm *PCM) []byte {
	t.Helper()
	data, err := Bytes(pcm)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// riff builds a file from chunks and fixes the RIFF size.
func riff(chunks ...[]byte) []byte {
	out := []byte("RIFF\x00\x00\x00\x00WAVE")
	for _, chunk := range chunks {
		out = append(out, chunk...)
	}
	binary.LittleEndian.PutUint32(out[4:], uint32(len(out)-8))
	return out
}

func chunk(id string, body []byte, pad bool) []byte {
	out := append([]byte(id), 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(out[4:], uint32(len(body)))
	out = append(out, body...)
	if pad && len(body)%2 == 1 {
		out = append(out, 0)
	}
	return out
}

func fmtBody(format, channels uint16, rate uint32, bits uint16, extra int) []byte {
	body := make([]byte, 16+extra)
	binary.LittleEndian.PutUint16(body[0:], format)
	binary.LittleEndian.PutUint16(body[2:], channels)
	binary.LittleEndian.PutUint32(body[4:], rate)
	binary.LittleEndian.PutUint32(body[8:], rate*uint32(channels)*uint32(bits)/8)
	binary.LittleEndian.PutUint16(body[12:], channels*bits/8)
	binary.LittleEndian.PutUint16(body[14:], bits)
	return body
}

func samplesBody(values ...int16) []byte {
	body := make([]byte, len(values)*2)
	for i, value := range values {
		binary.LittleEndian.PutUint16(body[i*2:], uint16(value))
	}
	return body
}

func TestDecodePreservesSignedPCMAcrossBufferBoundaries(t *testing.T) {
	pcm := signedPCM(2, 6006)
	data := mustBytes(t, pcm)
	withJunk := append([]byte(nil), data[:12]...)
	withJunk = append(withJunk, chunk("JUNK", []byte{1, 2, 3}, true)...)
	withJunk = append(withJunk, data[12:]...)
	for _, reader := range []io.Reader{bytes.NewReader(data), fragmentReader{bytes.NewReader(withJunk)}} {
		got, err := Decode(reader)
		if err != nil || !reflect.DeepEqual(got, pcm) {
			t.Fatalf("PCM changed after streaming decode: %v", err)
		}
	}
}

func TestDecodeChunkRules(t *testing.T) {
	format := chunk("fmt ", fmtBody(1, 1, 16000, 16, 0), true)
	t.Run("extended fmt and odd chunk padding", func(t *testing.T) {
		got, err := Decode(bytes.NewReader(riff(chunk("fmt ", fmtBody(1, 1, 16000, 16, 2), true),
			chunk("LIST", []byte{1}, true), chunk("data", samplesBody(5, -5), true))))
		if err != nil || !reflect.DeepEqual(got.Data, []int16{5, -5}) {
			t.Fatalf("got %v, %v", got, err)
		}
	})
	t.Run("data before fmt is ignored", func(t *testing.T) {
		_, err := Decode(bytes.NewReader(riff(chunk("data", samplesBody(1), true), format)))
		if err == nil {
			t.Fatal("data before fmt was used")
		}
		got, err := Decode(bytes.NewReader(riff(chunk("data", samplesBody(1), true), format, chunk("data", samplesBody(2), true))))
		if err != nil || !reflect.DeepEqual(got.Data, []int16{2}) {
			t.Fatalf("got %v, %v", got, err)
		}
	})
	t.Run("last data chunk wins", func(t *testing.T) {
		got, err := Decode(bytes.NewReader(riff(format, chunk("data", samplesBody(1, 2), true), chunk("data", samplesBody(3), true))))
		if err != nil || !reflect.DeepEqual(got.Data, []int16{3}) {
			t.Fatalf("got %v, %v", got, err)
		}
	})
	t.Run("empty data is valid", func(t *testing.T) {
		got, err := Decode(bytes.NewReader(riff(format, chunk("data", nil, true))))
		if err != nil || got.SampleRate != 16000 || got.Channels != 1 || len(got.Data) != 0 {
			t.Fatalf("got %v, %v", got, err)
		}
	})
}

func TestDecodeRejectsUnsupportedOrBrokenInput(t *testing.T) {
	format := chunk("fmt ", fmtBody(1, 2, 16000, 16, 0), true)
	valid := riff(format, chunk("data", samplesBody(1, 2), true))
	notWave := append([]byte(nil), valid...)
	copy(notWave[8:12], "AVI ")
	cases := map[string][]byte{
		"empty":            nil,
		"not riff":         append([]byte("RIFX"), valid[4:]...),
		"not wave":         notWave,
		"short fmt":        riff(chunk("fmt ", make([]byte, 14), true), chunk("data", samplesBody(1, 2), true)),
		"float format":     riff(chunk("fmt ", fmtBody(3, 1, 16000, 32, 0), true), chunk("data", make([]byte, 4), true)),
		"extensible":       riff(chunk("fmt ", fmtBody(0xfffe, 1, 16000, 16, 0), true), chunk("data", samplesBody(1), true)),
		"8-bit":            riff(chunk("fmt ", fmtBody(1, 1, 16000, 8, 0), true), chunk("data", []byte{1, 2}, true)),
		"odd data size":    riff(format, chunk("data", []byte{1, 2, 3}, true)),
		"missing data":     riff(format),
		"missing fmt":      riff(chunk("data", samplesBody(1), true)),
		"zero rate":        riff(chunk("fmt ", fmtBody(1, 1, 0, 16, 0), true), chunk("data", samplesBody(1), true)),
		"zero channels":    riff(chunk("fmt ", fmtBody(1, 0, 16000, 16, 0), true), chunk("data", samplesBody(1), true)),
		"channel misalign": riff(format, chunk("data", samplesBody(1, 2, 3), true)),
		"truncated data":   valid[:len(valid)-1],
		"missing pad":      riff(format, chunk("data", samplesBody(1, 2), true), chunk("LIST", []byte{1}, false)),
		"partial chunk":    append(append([]byte(nil), valid...), []byte("LI")...),
		"partial size":     append(append([]byte(nil), valid...), []byte("LIST\x01")...),
	}
	for name, data := range cases {
		if _, err := Decode(bytes.NewReader(data)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestDecodeDoesNotTrustHugeDataSize(t *testing.T) {
	data := riff(chunk("fmt ", fmtBody(1, 1, 16000, 16, 0), true))
	data = append(data, []byte("data\xfe\xff\xff\xff")...)
	data = append(data, samplesBody(1, 2, 3)...)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, err := Decode(bytes.NewReader(data))
	runtime.ReadMemStats(&after)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v", err)
	}
	if grown := after.TotalAlloc - before.TotalAlloc; grown > 64<<20 {
		t.Fatalf("allocated %d bytes for a 3-sample file", grown)
	}
}

func TestEncodeMatchesBytesAndRoundTrips(t *testing.T) {
	for _, pcm := range []*PCM{signedPCM(1, 0), signedPCM(1, 1), signedPCM(2, 20000)} {
		original := append([]int16(nil), pcm.Data...)
		want := mustBytes(t, pcm)
		var out bytes.Buffer
		if err := Encode(&out, pcm); err != nil || !bytes.Equal(out.Bytes(), want) {
			t.Fatalf("Encode and Bytes differ: %v", err)
		}
		if len(want) != 44+len(pcm.Data)*2 || binary.LittleEndian.Uint32(want[4:]) != uint32(len(want)-8) {
			t.Fatal("wrong RIFF size")
		}
		got, err := Decode(bytes.NewReader(want))
		if err != nil || got.SampleRate != pcm.SampleRate || got.Channels != pcm.Channels ||
			!slices.Equal(got.Data, original) {
			t.Fatalf("round trip changed PCM: %v", err)
		}
		if !slices.Equal(pcm.Data, original) {
			t.Fatal("encoding modified the input")
		}
	}
}

type failingWriter struct{ after int }

func (w *failingWriter) Write(data []byte) (int, error) {
	if w.after <= 0 {
		return 0, errors.New("write failed")
	}
	w.after--
	return len(data), nil
}

func TestEncodeValidatesFormatAndReportsWriteErrors(t *testing.T) {
	for name, pcm := range map[string]*PCM{
		"zero rate":     {SampleRate: 0, Channels: 1},
		"zero channels": {SampleRate: 16000, Channels: 0},
		"too many":      {SampleRate: 16000, Channels: math.MaxUint16},
		"byte rate":     {SampleRate: math.MaxInt32, Channels: 4},
	} {
		if _, err := Bytes(pcm); err == nil {
			t.Errorf("%s accepted by Bytes", name)
		}
		if err := Encode(io.Discard, pcm); err == nil {
			t.Errorf("%s accepted by Encode", name)
		}
	}
	pcm := signedPCM(1, 10000)
	for after := 0; after < 3; after++ {
		if err := Encode(&failingWriter{after: after}, pcm); err == nil {
			t.Fatalf("write error after %d writes was ignored", after)
		}
	}
}
