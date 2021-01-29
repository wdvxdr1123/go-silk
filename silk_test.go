package silk

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestDecodeSilkBuffToWave(t *testing.T) {
	b, err := ioutil.ReadFile("test1.silk")
	assert.Nil(t, err)
	dst, err := DecodeSilkBuffToWave(b, 8000)
	assert.Nil(t, err)
	err = ioutil.WriteFile("test1.wav", dst, 0666)
	assert.Nil(t, err)
}

func TestDecodePcmBuffToSilk(t *testing.T) {
	b, err := ioutil.ReadFile("test.pcm")
	assert.Nil(t, err)
	dst, err := EncodeWavBuffToSilk(b, 24000, true)
	assert.Nil(t, err)
	err = ioutil.WriteFile("test1.silk", dst, 0666)
	assert.Nil(t, err)
}
