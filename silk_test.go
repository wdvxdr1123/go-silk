package silk

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestDecodeSilkBuffToWave(t *testing.T) {
	b, err := ioutil.ReadFile("test.silk")
	assert.Nil(t, err)
	dst, err := DecodeSilkBuffToWave(b, 8000)
	assert.Nil(t, err)
	err = ioutil.WriteFile("test.wav", dst, 0666)
	assert.Nil(t, err)
}
