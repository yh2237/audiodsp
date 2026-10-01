// Package wav reads and writes 16-bit integer PCM RIFF/WAVE data.
package wav

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"slices"
)

// PCM is interleaved 16-bit PCM. Data holds Channels samples per frame.
type PCM struct {
	SampleRate int
	Channels   int
	Data       []int16
}

const headerSize = 44

// Initial capacity limit for a data chunk. Larger chunks grow while reading so a
// corrupt size field cannot force a huge allocation before any samples arrive.
const maxInitialSamples = 16 << 20

// Decode reads a RIFF/WAVE stream with a PCM (format 1) fmt chunk and 16-bit samples.
// Unknown chunks are skipped, odd-sized chunks must carry their pad byte, and a data
// chunk that appears before fmt is ignored. When several data chunks follow fmt, the
// last one is returned.
func Decode(input io.Reader) (*PCM, error) {
	reader := bufio.NewReader(input)
	var header [12]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return nil, err
	}
	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return nil, errors.New("wav: invalid header")
	}

	var (
		fmtFound      bool
		dataFound     bool
		sampleRate    uint32
		channels      uint16
		bitsPerSample uint16
		samples       []int16
	)
	for {
		var chunk [8]byte
		if _, err := io.ReadFull(reader, chunk[:]); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		id := string(chunk[:4])
		size := binary.LittleEndian.Uint32(chunk[4:])
		switch {
		case id == "fmt ":
			if size < 16 {
				return nil, errors.New("wav: invalid fmt chunk")
			}
			var format [16]byte
			if _, err := io.ReadFull(reader, format[:]); err != nil {
				return nil, eof(err)
			}
			if binary.LittleEndian.Uint16(format[0:]) != 1 {
				return nil, errors.New("wav: only PCM format is supported")
			}
			channels = binary.LittleEndian.Uint16(format[2:])
			sampleRate = binary.LittleEndian.Uint32(format[4:])
			bitsPerSample = binary.LittleEndian.Uint16(format[14:])
			if err := skip(reader, int64(size)-16); err != nil {
				return nil, err
			}
			fmtFound = true
		case id == "data" && fmtFound:
			if bitsPerSample != 16 {
				return nil, errors.New("wav: only 16-bit PCM is supported")
			}
			if size%2 != 0 {
				return nil, errors.New("wav: invalid 16-bit PCM data size")
			}
			var err error
			if samples, err = readSamples(reader, int(size/2)); err != nil {
				return nil, err
			}
			dataFound = true
		default:
			if err := skip(reader, int64(size)); err != nil {
				return nil, err
			}
		}
		if size%2 == 1 {
			if _, err := reader.ReadByte(); err != nil {
				return nil, eof(err)
			}
		}
	}

	if !fmtFound || !dataFound {
		return nil, errors.New("wav: missing fmt or data chunk")
	}
	if sampleRate == 0 || channels == 0 || sampleRate > math.MaxInt32 {
		return nil, errors.New("wav: sample rate and channels must be positive")
	}
	if len(samples)%int(channels) != 0 {
		return nil, errors.New("wav: sample data is not aligned to channels")
	}
	return &PCM{SampleRate: int(sampleRate), Channels: int(channels), Data: samples}, nil
}

// readSamples decodes directly from the bufio buffer without a full-size byte copy.
func readSamples(reader *bufio.Reader, count int) ([]int16, error) {
	samples := make([]int16, 0, min(count, maxInitialSamples))
	for len(samples) < count {
		n := min(count-len(samples), reader.Size()/2)
		data, err := reader.Peek(n * 2)
		if err != nil {
			return nil, eof(err)
		}
		start := len(samples)
		samples = slices.Grow(samples, n)[:start+n]
		dst := samples[start:]
		for i := range dst {
			dst[i] = int16(binary.LittleEndian.Uint16(data[i*2:]))
		}
		if _, err := reader.Discard(n * 2); err != nil {
			return nil, err
		}
	}
	return samples, nil
}

func skip(reader *bufio.Reader, n int64) error {
	if n <= 0 {
		return nil
	}
	if _, err := io.CopyN(io.Discard, reader, n); err != nil {
		return eof(err)
	}
	return nil
}

// A stream that ends inside a chunk is truncated, not a clean end of input.
func eof(err error) error {
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
}

func header(pcm *PCM) ([headerSize]byte, error) {
	var out [headerSize]byte
	if pcm.SampleRate <= 0 || pcm.Channels <= 0 || pcm.Channels > math.MaxUint16/2 {
		return out, errors.New("wav: sample rate and channels must be positive")
	}
	byteRate := uint64(pcm.SampleRate) * uint64(pcm.Channels) * 2
	dataSize := uint64(len(pcm.Data)) * 2
	if byteRate > math.MaxUint32 || dataSize > math.MaxUint32-(headerSize-8) {
		return out, errors.New("wav: data does not fit a RIFF/WAVE file")
	}
	copy(out[0:4], "RIFF")
	binary.LittleEndian.PutUint32(out[4:], uint32(dataSize+headerSize-8))
	copy(out[8:16], "WAVEfmt ")
	binary.LittleEndian.PutUint32(out[16:], 16)
	binary.LittleEndian.PutUint16(out[20:], 1)
	binary.LittleEndian.PutUint16(out[22:], uint16(pcm.Channels))
	binary.LittleEndian.PutUint32(out[24:], uint32(pcm.SampleRate))
	binary.LittleEndian.PutUint32(out[28:], uint32(byteRate))
	binary.LittleEndian.PutUint16(out[32:], uint16(pcm.Channels*2))
	binary.LittleEndian.PutUint16(out[34:], 16)
	copy(out[36:40], "data")
	binary.LittleEndian.PutUint32(out[40:], uint32(dataSize))
	return out, nil
}

// Encode writes a 44-byte PCM header followed by the samples. Samples are converted
// through a fixed buffer, so the output is not built in memory first.
func Encode(output io.Writer, pcm *PCM) error {
	head, err := header(pcm)
	if err != nil {
		return err
	}
	if _, err := output.Write(head[:]); err != nil {
		return err
	}
	var buffer [8192]byte
	for data := pcm.Data; len(data) > 0; {
		n := min(len(data), len(buffer)/2)
		for i, sample := range data[:n] {
			binary.LittleEndian.PutUint16(buffer[i*2:], uint16(sample))
		}
		if _, err := output.Write(buffer[:n*2]); err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}

// Bytes returns the encoded file in a newly allocated slice of exactly its size.
func Bytes(pcm *PCM) ([]byte, error) {
	head, err := header(pcm)
	if err != nil {
		return nil, err
	}
	out := make([]byte, headerSize+len(pcm.Data)*2)
	copy(out, head[:])
	for i, sample := range pcm.Data {
		binary.LittleEndian.PutUint16(out[headerSize+i*2:], uint16(sample))
	}
	return out, nil
}
