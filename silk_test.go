package silk

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDecodeSilkBuffToPcm(t *testing.T) {
	b, err := ioutil.ReadFile("test3.silk")
	assert.Nil(t, err)
	dst, err := DecodeSilkBuffToPcm(b, 24000)
	assert.Nil(t, err)
	err = ioutil.WriteFile("test3.pcm", dst, 0666)
	assert.Nil(t, err)
}

func TestDecodePcmBuffToSilk(t *testing.T) {
	b, err := ioutil.ReadFile("xcz.pcm")
	assert.Nil(t, err)
	start := time.Now() // 5194447
	dst, err := EncodePcmBuffToSilk(b, 24000, 24000, true)
	fmt.Println(time.Since(start).Microseconds())
	assert.Nil(t, err)
	err = ioutil.WriteFile("test4.silk", dst, 0666)
	assert.Nil(t, err)
}

func BenchmarkPcm(b *testing.B) {
	p, _ := ioutil.ReadFile("xcz.pcm")
	b.SetBytes(int64(len(p)))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		EncodePcmBuffToSilk(p, 24000, 24000, true)
	}
}
