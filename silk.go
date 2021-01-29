package silk

import (
	"bytes"
	"encoding/binary"
	"errors"
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
