package silk

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
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
	b, err := ioutil.ReadFile("test3.pcm")
	assert.Nil(t, err)
	dst, err := EncodePcmBuffToSilk(b, 24000, 24000, true)
	assert.Nil(t, err)
	err = ioutil.WriteFile("test4.silk", dst, 0666)
	assert.Nil(t, err)
}
