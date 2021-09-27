package exporter

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReadCustomPoliciesFromLocalRepository(t *testing.T) {
	if SkipTest() {
		return
	}

	policies, err := ReadCustomPoliciesFromLocalRepository("/tmp/ACF3.Modular")
	assert.NoError(t, err)
	assert.NotEmpty(t, policies)

	_ = PrettyPrint(policies)
}
