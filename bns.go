package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"
)

// Conversion of bns.cs to go

// BNS public type
type BNS struct {
	bnsHeader       BNSHeader
	bnsInfo         BNSInfo
	bnsData         BNSData
	rlSamples       [2][2]int
	tlSamples       [2]int
	defTbl          [16]int
	pHist1          [2]int
	pHist2          [2]int
	tempSampleCount int
	waveFile        []byte
	loopFromWave    bool
	converted       bool
	toMono          bool
}

// Wave data returned by BnsToWave
type Wave struct {
	DataFormat  int
	NumChannels int
	SampleRate  int
	BitDepth    int
	SampleData  []byte
	NumLoops    int
	LoopStart   int
}

func NewFromFile(path string) (*BNS, error) {
	b := &BNS{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	b.waveFile = data
	b.defTbl = [16]int{1820, -856, 3238, -1514, 2333, -550, 3336, -1376, 2444, -949, 3666, -1764, 2654, -701, 3420, -1398}
	return b, nil
}

func NewFromBytes(data []byte, loopFromWave bool) *BNS {
	b := &BNS{waveFile: data, loopFromWave: loopFromWave}
	b.defTbl = [16]int{1820, -856, 3238, -1514, 2333, -550, 3336, -1376, 2444, -949, 3666, -1764, 2654, -701, 3420, -1398}
	return b
}

func NewBNSFromWAVBytes(wav []byte) (*BNS, error) {
	if _, err := NewWaveFromBytes(wav); err != nil {
		return nil, err
	}
	return NewFromBytes(wav, true), nil
}

func (b *BNS) SetStereoToMono(v bool) { b.toMono = v }

func (b *BNS) Convert() error { return b.convertInternal(b.waveFile, b.loopFromWave) }

func (b *BNS) ToBytes() ([]byte, error) {
	if !b.converted {
		if err := b.convertInternal(b.waveFile, b.loopFromWave); err != nil {
			return nil, err
		}
	}
	buf := &bytes.Buffer{}
	if err := b.bnsHeader.Write(buf); err != nil {
		return nil, err
	}
	if err := b.bnsInfo.Write(buf); err != nil {
		return nil, err
	}
	if err := b.bnsData.Write(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (b *BNS) convertInternal(waveFile []byte, loopFromWave bool) error {
	wave, err := NewWaveFromBytes(waveFile)
	if err != nil {
		return err
	}
	numLoops := wave.NumLoops
	loopStart := wave.LoopStart
	b.bnsInfo.ChannelCount = byte(wave.NumChannels)
	b.bnsInfo.SampleRate = uint16(wave.SampleRate)
	if b.bnsInfo.ChannelCount > 2 || b.bnsInfo.ChannelCount < 1 {
		return errors.New("unsupported amount of channels")
	}
	if wave.BitDepth != 16 {
		return errors.New("only 16bit wave files are supported")
	}
	b.bnsData.Data = b.encode(wave.SampleData)
	if b.bnsInfo.ChannelCount == 1 {
		// INFO chunk total size (including "INFO" and size uint32) for mono
		b.bnsInfo.Size = 96
		b.bnsInfo.Channel1StartOffset = 28
		b.bnsInfo.Channel2StartOffset = 0
		b.bnsInfo.Channel1Start = 40
		b.bnsInfo.Coefficients1Offset = 0
	}
	b.bnsData.Size = uint32(len(b.bnsData.Data) + 8)
	// Header setup
	b.bnsHeader.Size = 32
	b.bnsHeader.ChunkCount = 2
	b.bnsHeader.InfoOffset = 32
	b.bnsHeader.InfoLength = b.bnsInfo.Size
	b.bnsHeader.DataOffset = b.bnsHeader.InfoOffset + b.bnsHeader.InfoLength
	b.bnsHeader.DataLength = b.bnsData.Size
	b.bnsHeader.FileSize = uint32(b.bnsHeader.Size) + b.bnsHeader.InfoLength + b.bnsHeader.DataLength
	b.bnsHeader.Flags = 0xfeff0100
	if loopFromWave && numLoops == 1 && loopStart != -1 {
		b.bnsInfo.LoopStart = uint32(loopStart)
		b.bnsInfo.HasLoop = 1
	}
	b.bnsInfo.LoopEnd = uint32(b.tempSampleCount)
	for i := 0; i < 16; i++ {
		b.bnsInfo.Coefficients1[i] = int16(b.defTbl[i])
		if b.bnsInfo.ChannelCount == 2 {
			b.bnsInfo.Coefficients2[i] = int16(b.defTbl[i])
		}
	}
	b.converted = true
	return nil
}

func NewWaveFromBytes(b []byte) (*Wave, error) {
	if len(b) < 12 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return nil, errors.New("not a WAV file")
	}
	w := &Wave{}
	off := 12
	for off+8 <= len(b) {
		id := string(b[off : off+4])
		off += 4
		size := int(binary.LittleEndian.Uint32(b[off : off+4]))
		off += 4
		if off+size > len(b) {
			size = len(b) - off
		}
		switch id {
		case "fmt ":
			if size < 16 {
				return nil, errors.New("invalid fmt chunk")
			}
			w.DataFormat = int(binary.LittleEndian.Uint16(b[off : off+2]))
			w.NumChannels = int(binary.LittleEndian.Uint16(b[off+2 : off+4]))
			w.SampleRate = int(binary.LittleEndian.Uint32(b[off+4 : off+8]))
			w.BitDepth = int(binary.LittleEndian.Uint16(b[off+14 : off+16]))
		case "data":
			w.SampleData = make([]byte, size)
			copy(w.SampleData, b[off:off+size])
		case "smpl":
			if size >= 36 {
				w.NumLoops = int(binary.LittleEndian.Uint32(b[off+28 : off+32]))
				if w.NumLoops > 0 && size >= 60 {
					w.LoopStart = int(binary.LittleEndian.Uint32(b[off+44 : off+48]))
				}
			}
		}
		off += size
	}
	return w, nil
}

func (b *BNS) encode(inputFrames []byte) []byte {
	frameSize := func() int {
		if b.bnsInfo.ChannelCount == 2 {
			return 4
		}
		return 2
	}()
	samples := len(inputFrames) / frameSize
	b.tempSampleCount = samples
	num1 := samples % 14
	if num1 != 0 {
		inputFrames = append(inputFrames, make([]byte, (14-num1)*frameSize)...)
	}
	num2 := len(inputFrames) / frameSize
	num3 := (num2 + 13) / 14
	intList1 := make([]int, 0, num2)
	intList2 := make([]int, 0, num2)
	off := 0
	if b.toMono && b.bnsInfo.ChannelCount == 2 {
		b.bnsInfo.ChannelCount = 1
	} else if b.toMono {
		b.toMono = false
	}
	for i := 0; i < num2; i++ {
		intList1 = append(intList1, int(int16(binary.LittleEndian.Uint16(inputFrames[off:off+2]))))
		off += 2
		if b.bnsInfo.ChannelCount == 2 || b.toMono {
			intList2 = append(intList2, int(int16(binary.LittleEndian.Uint16(inputFrames[off:off+2]))))
			off += 2
		}
	}
	out := make([]byte, 0)
	for idx := 0; idx < num3; idx++ {
		var inputBuffer [14]int
		for j := 0; j < 14; j++ {
			inputBuffer[j] = intList1[idx*14+j]
		}
		numArray2 := b.repackAdpcm(0, b.defTbl[:], inputBuffer[:])
		out = append(out, numArray2...)
		if b.bnsInfo.ChannelCount == 2 {
			for j := 0; j < 14; j++ {
				inputBuffer[j] = intList2[idx*14+j]
			}
			numArray3 := b.repackAdpcm(1, b.defTbl[:], inputBuffer[:])
			out = append(out, numArray3...)
		}
	}
	b.bnsInfo.LoopEnd = uint32(num3 * 7)
	return out
}

func (b *BNS) repackAdpcm(index int, table []int, inputBuffer []int) []byte {
	res := make([]byte, 8)
	bestErr := math.Inf(1)
	bestTl := [2]int{}
	for tableIndex := 0; tableIndex < 8; tableIndex++ {
		cand, outErr := b.compressAdpcm(index, table, tableIndex, inputBuffer)
		if outErr < bestErr {
			bestErr = outErr
			copy(res, cand)
			bestTl = b.tlSamples
		}
	}
	b.rlSamples[index][0] = bestTl[0]
	b.rlSamples[index][1] = bestTl[1]
	return res
}

func (b *BNS) compressAdpcm(index int, table []int, tableIndex int, inputBuffer []int) ([]byte, float64) {
	numArray := make([]byte, 8)
	num2 := table[2*tableIndex]
	num3 := table[2*tableIndex+1]
	stdExponent := b.determineStdExponent(index, table, tableIndex, inputBuffer)
	for stdExponent <= 15 {
		for k := range numArray {
			numArray[k] = 0
		}
		numArray[0] = byte(stdExponent | tableIndex<<4)
		for i := 0; i < 2; i++ {
			b.tlSamples[i] = b.rlSamples[index][i]
		}
		num1 := 0
		broken := false
		for i := 0; i < 14; i++ {
			num5 := (b.tlSamples[1]*num2 + b.tlSamples[0]*num3) >> 11
			input1 := (inputBuffer[i] - num5) >> stdExponent
			if input1 <= 7 && input1 >= -8 {
				num6 := clamp(input1, -8, 7)
				if (i & 1) == 0 {
					numArray[i/2+1] = byte(num6 << 4)
				} else {
					numArray[i/2+1] = numArray[i/2+1] | byte(num6&15)
				}
				input2 := num5 + (num6 << stdExponent)
				b.tlSamples[0] = b.tlSamples[1]
				b.tlSamples[1] = clamp(input2, -32768, 32767)
				num1 += (b.tlSamples[1] - inputBuffer[i]) * (b.tlSamples[1] - inputBuffer[i])
			} else {
				stdExponent++
				broken = true
				break
			}
		}
		if !broken {
			return numArray, float64(num1)
		}
	}
	return numArray, float64(1e18)
}

func (b *BNS) determineStdExponent(index int, table []int, tableIndex int, inputBuffer []int) int {
	num2 := table[2*tableIndex]
	num3 := table[2*tableIndex+1]
	numArray := [2]int{b.rlSamples[index][0], b.rlSamples[index][1]}
	max := 0
	for i := 0; i < 14; i++ {
		num4 := (numArray[1]*num2 + numArray[0]*num3) >> 11
		d := inputBuffer[i] - num4
		if d > max {
			max = d
		}
		numArray[0] = numArray[1]
		numArray[1] = inputBuffer[i]
	}
	return findExponent(float64(max))
}

func findExponent(res float64) int {
	n := 0
	for res > 7.5 || res < -8.5 {
		res /= 2
		n++
	}
	return n
}
func clamp(x, lo, hi int) int {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

type BNSData struct {
	Data []byte
	Size uint32
}

func (d *BNSData) Write(w io.Writer) error {
	if _, err := w.Write([]byte("DATA")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, d.Size); err != nil {
		return err
	}
	_, err := w.Write(d.Data)
	return err
}

type BNSHeader struct {
	Flags, FileSize                                uint32
	Size, ChunkCount                               uint16
	InfoOffset, InfoLength, DataOffset, DataLength uint32
}

func (h *BNSHeader) Write(w io.Writer) error {
	if _, err := w.Write([]byte("BNS ")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.Flags); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.FileSize); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.Size); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.ChunkCount); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.InfoOffset); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.InfoLength); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.DataOffset); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.DataLength); err != nil {
		return err
	}
	return nil
}

type BNSInfo struct {
	HasLoop                                                                      byte
	ChannelCount                                                                 byte
	Size                                                                         uint32
	SampleRate                                                                   uint16
	LoopStart                                                                    uint32
	LoopEnd                                                                      uint32
	Channel1StartOffset, Channel2StartOffset, Channel1Start, Coefficients1Offset uint32
	Channel2Start, Coefficients2Offset                                           uint32
	Coefficients1                                                                [16]int16
	Coefficients2                                                                [16]int16
}

func (i *BNSInfo) Write(w io.Writer) error {
	payload := &bytes.Buffer{}
	_ = binary.Write(payload, binary.BigEndian, byte(0))
	_ = binary.Write(payload, binary.BigEndian, i.HasLoop)
	_ = binary.Write(payload, binary.BigEndian, i.ChannelCount)
	_ = binary.Write(payload, binary.BigEndian, byte(0))
	_ = binary.Write(payload, binary.BigEndian, i.SampleRate)
	_ = binary.Write(payload, binary.BigEndian, uint16(0))
	_ = binary.Write(payload, binary.BigEndian, i.LoopStart)
	_ = binary.Write(payload, binary.BigEndian, i.LoopEnd)
	_ = binary.Write(payload, binary.BigEndian, uint32(24))
	_ = binary.Write(payload, binary.BigEndian, uint32(0))
	_ = binary.Write(payload, binary.BigEndian, i.Channel1StartOffset)
	_ = binary.Write(payload, binary.BigEndian, i.Channel2StartOffset)
	_ = binary.Write(payload, binary.BigEndian, i.Channel1Start)
	_ = binary.Write(payload, binary.BigEndian, i.Coefficients1Offset)
	if i.ChannelCount == 2 {
		_ = binary.Write(payload, binary.BigEndian, uint32(0))
		_ = binary.Write(payload, binary.BigEndian, i.Channel2Start)
		_ = binary.Write(payload, binary.BigEndian, i.Coefficients2Offset)
		_ = binary.Write(payload, binary.BigEndian, uint32(0))
		for _, v := range i.Coefficients1 {
			_ = binary.Write(payload, binary.BigEndian, v)
		}
		for _, v := range i.Coefficients2 {
			_ = binary.Write(payload, binary.BigEndian, v)
		}
	} else {
		for _, v := range i.Coefficients1 {
			_ = binary.Write(payload, binary.BigEndian, v)
		}
	}
	// write header + size
	if _, err := w.Write([]byte("INFO")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, i.Size); err != nil {
		return err
	}
	if _, err := w.Write(payload.Bytes()); err != nil {
		return err
	}
	if want := int(i.Size) - 8 - payload.Len(); want > 0 {
		pad := make([]byte, want)
		if _, err := w.Write(pad); err != nil {
			return err
		}
	}
	return nil
}
