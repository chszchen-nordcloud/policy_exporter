package exporter

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestReadPolicyDefinitionFromExcel(t *testing.T) {
	if SkipTest() {
		return
	}
	base := TestResourceDir()
	result, err := ReadPolicyDefinitionFromExcel(
		filepath.Join(base, "Azure Cloud Foundation - Baseline Policies.xlsx"), "", "", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, result.BuiltInPolicies)
	assert.NotEmpty(t, result.CustomPolicies)
	assert.NotEmpty(t, result.ASCPolicySetParameters)
	_ = PrettyPrint(result)
}
