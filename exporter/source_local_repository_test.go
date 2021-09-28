package exporter

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReadCustomPoliciesFromLocalRepository(t *testing.T) {
	if SkipTest() {
		return
	}

	policies, err := ReadCustomPoliciesFromLocalRepository("/Users/chszchen/Home/Temp/azure/ACF3.Modular")
	assert.NoError(t, err)
	assert.NotEmpty(t, policies)

	println(len(policies))
	_ = PrettyPrint(policies)
}
