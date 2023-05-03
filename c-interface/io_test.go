package c_interface

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"strings"
	"testing"
)

func TestStringBuilder(t *testing.T) {
	content := []byte("1234567890")
	source := bytes.NewBuffer(content)
	target := &strings.Builder{}

	assert := assert.New(t)
	w, e := io.Copy(target, source)
	assert.Nil(e)
	assert.Equal(int64(len(content)), w)
}
