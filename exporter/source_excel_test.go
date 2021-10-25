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
	config := getConfigForTest(t)
	result, err := ReadPolicyDefinitionFromExcel(
		filepath.Join(base, "Azure Cloud Foundation - Baseline Policies - WithUserInputs.xlsx"),
		config.ManagementGroups,
		config.Subscriptions,
	)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.BuiltInPolicies)
	assert.NotEmpty(t, result.CustomPolicies)
	assert.NotEmpty(t, result.ASCPolicySetParameters)

	customPolicy := result.CustomPolicies[0]
	assert.NotEqual(t, "", customPolicy.DisplayName)

	_ = PrettyPrint(result)
}
