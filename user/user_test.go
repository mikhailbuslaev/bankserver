package user

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func Test_Create(t *testing.T) {
	assert := assert.New(t)
	want := &User{Id:"1", AuthKey:"1", Balance:0.0}
	got := Create("1", "1", 0.00)
	assert.Equal(want, got, "they should be equal")
}