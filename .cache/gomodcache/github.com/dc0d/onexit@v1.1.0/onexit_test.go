package onexit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeferred(t *testing.T) {
	assert := assert.New(t)
	d := newDeferred()

	var order []int

	// the functions are being registered from lowest priority
	// to highest.
	for i := -10; i <= 10; i++ {
		i := i
		d.Register(func() {
			order = append(order, i)
		}, i)
	}

	go d.Cleanup()
	go d.Cleanup()
	go d.Cleanup()

	<-d.Done()

	// the highest priority will run first.
	for i := 0; i < 21; i++ {
		assert.Equal(20-i-10, order[i])
	}
}

func TestRegister(t *testing.T) {
	assert := assert.New(t)

	var order []int

	// the functions are being registered from lowest priority
	// to highest.
	for i := -10; i <= 10; i++ {
		i := i
		Register(func() {
			order = append(order, i)
		}, i)
	}

	go forceExit(0, false)

	<-Done()

	// the highest priority will run first.
	for i := 0; i < 21; i++ {
		assert.Equal(20-i-10, order[i])
	}
}
