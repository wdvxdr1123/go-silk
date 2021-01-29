package silk

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/orcaman/writerseeker"
	"io"
	"io/ioutil"
	"modernc.org/libc"
	"modernc.org/libc/sys/types"
	"silk/internal"
	"unsafe"
)

var (
	ErrInvalid    = errors.New("not a silk stream")
	ErrCodecError = errors.New("codec error")
)

func DecodeSilkBuffToWave(src []byte, sampleRate int) (dst []byte, err error) {
	var tls = libc.NewTLS()
	reader := bytes.NewBuffer(src)
	f, err := reader.ReadByte()
	if err != nil {
		return
	}
	header := make([]byte, 9)
	var n int
	if f == 2 {
		n, err = reader.Read(header)
		if err != nil {
			return
		}
		if n != 9 {
			err = ErrInvalid
			return
		}
		if string(header) != "#!SILK_V3" {
			err = ErrInvalid
			return
		}
	} else if f == '#' {
		n, err = reader.Read(header)
		if err != nil {
			return
		}
		if n != 8 {
			err = ErrInvalid
			return
		}
		if string(header) != "!SILK_V3" {
			err = ErrInvalid
			return
		}
	} else {
		err = ErrInvalid
		return
	}
	var decControl internal.SKP_SILK_SDK_DecControlStruct
	decControl.FAPI_sampleRate = int32(sampleRate)
	decControl.FframesPerPacket = 1
	var decSize int32
	internal.SKP_Silk_SDK_Get_Decoder_Size(tls, uintptr(unsafe.Pointer(&decSize)))
	dec := libc.Xmalloc(tls, types.Size_t(decSize))
	defer libc.Xfree(tls, dec)
	if internal.SKP_Silk_init_decoder(tls, dec) != 0 {
		err = ErrCodecError
		return
	}
	// 40ms
	frameSize := sampleRate / 1000 * 40
	in := make([]byte, frameSize)
	buf := make([]int16, frameSize)
	out := &writerseeker.WriterSeeker{}
	enc := wav.NewEncoder(out, sampleRate, 16, 1, 1)
	audioBuf := &audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: 1,
			SampleRate:  sampleRate,
		},
	}
	for {
		var nByte int16
		err = binary.Read(reader, binary.LittleEndian, &nByte)
		if err != nil {
			if err == io.EOF {
				break
			}
			return
		}
		if int(nByte) > frameSize {
			err = ErrInvalid
			return
		}
		n, err = reader.Read(in[:nByte])
		if err != nil {
			return
		}
		if n != int(nByte) {
			err = ErrInvalid
			return
		}
		internal.SKP_Silk_SDK_Decode(tls, dec, uintptr(unsafe.Pointer(&decControl)), 0,
			uintptr(unsafe.Pointer(&in[0])), int32(n),
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(&nByte)))

		for _, w := range buf[:int(nByte)] {
			audioBuf.Data = append(audioBuf.Data, int(w))
		}
	}
	if err = enc.Write(audioBuf); err != nil {
		return
	}
	if err = enc.Close(); err != nil {
		return
	}
	dst, err = ioutil.ReadAll(out.Reader())
	return
}

func EncodeWavBuffToSilk(src []byte, bitRate int, tencent bool) (dst []byte, err error) {
	var tls = libc.NewTLS()
	var reader = bytes.NewBuffer(src)
	var encControl internal.SKP_SILK_SDK_EncControlStruct
	var encStatus internal.SKP_SILK_SDK_EncControlStruct
	var smplsSinceLastPacket int32
	var packetSizeMs = int32(20)
	const (
		ApiFsHz        = int32(24000)
		targetRateBps  = 24000
		packetLossPerc = int32(0)
	)
	{ // default setting
		encControl.FAPI_sampleRate = 24000
		encControl.FmaxInternalSampleRate = 24000
		encControl.FpacketSize = (packetSizeMs * ApiFsHz) / 1000
		encControl.FpacketLossPercentage = packetLossPerc
		encControl.FuseDTX = 0
		encControl.Fcomplexity = 2
		encControl.FbitRate = int32(targetRateBps)
		encControl.FAPI_sampleRate = 24000
		encControl.FbitRate = int32(bitRate)
	}
	var encSizeBytes int32
	ret := internal.SKP_Silk_SDK_Get_Encoder_Size(tls, uintptr(unsafe.Pointer(&encSizeBytes)))
	if ret != 0 {
		return nil, fmt.Errorf("SKP_Silk_create_encoder returned %d", ret)
	}
	psEnc := libc.Xmalloc(tls, types.Size_t(encSizeBytes))
	defer libc.Xfree(tls, psEnc)
	ret = internal.SKP_Silk_SDK_InitEncoder(tls, psEnc, uintptr(unsafe.Pointer(&encStatus)))
	if ret != 0 {
		return nil, fmt.Errorf("SKP_Silk_reset_encoder returned %d", ret)
	}
	const frameSize = 24000 / 1000 * 40
	var (
		nBytes  = int16(250 * 5)
		in      = make([]byte, frameSize)
		payload = make([]byte, nBytes)
		out     = writerseeker.WriterSeeker{}
	)
	smplsSinceLastPacket = 0
	if tencent {
		_, _ = out.Write([]byte("\x02#!SILK_V3"))
	} else {
		_, _ = out.Write([]byte("#!SILK_V3"))
	}
	defer func() {
		dst, _ = ioutil.ReadAll(out.BytesReader())
	}()
	var counter int
	for {
		counter, err = reader.Read(in)
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return
		}
		if counter < frameSize {
			break
		}
		nBytes = 250 * 5
		ret = internal.SKP_Silk_SDK_Encode(
			tls,
			psEnc,
			uintptr(unsafe.Pointer(&encControl)),
			uintptr(unsafe.Pointer(&in[0])),
			int32(counter),
			uintptr(unsafe.Pointer(&payload[0])),
			uintptr(unsafe.Pointer(&nBytes)),
		)

		if ret != 0 {
			return nil, fmt.Errorf("SKP_Silk_Encode returned %d", ret)
		}

		packetSizeMs = (1000 * encControl.FpacketSize) / encControl.FAPI_sampleRate
		smplsSinceLastPacket += int32(counter)
		if ((1000 * smplsSinceLastPacket) / ApiFsHz) == packetSizeMs {
			var nByte = make([]byte, 2)
			binary.LittleEndian.PutUint16(nByte, uint16(nBytes))
			_, _ = out.Write(nByte[:2])
			_, _ = out.Write(payload[:nBytes])
			smplsSinceLastPacket = 0
		}
	}
	if !tencent {
		var b []byte
		binary.LittleEndian.PutUint16(b, ^uint16(0)) // -1
		_, _ = out.Write(b)
	}
	return
}
