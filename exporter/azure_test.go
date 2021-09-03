package exporter

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_foo(t *testing.T) {
	az, err := NewAzureAPI("")
	assert.NoError(t, err)

	err = az.Foo(context.Background())
	assert.NoError(t, err)
}
