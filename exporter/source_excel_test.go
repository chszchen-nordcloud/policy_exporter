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

	var expectedPolicy Policy
	for _, policy := range result.BuiltInPolicies {
		if policy.Justification != "" {
			expectedPolicy = policy
			break
		}
	}
	assert.NotEqual(t, "", expectedPolicy.Justification)

	customPolicy := result.CustomPolicies[0]
	assert.NotEqual(t, "", customPolicy.DisplayName)
	assert.NotEqual(t, "", customPolicy.Description)

	var expectedParameter PolicyParameter
	for _, param := range result.ASCPolicySetParameters {
		if param.Justification != "" && param.CostImpact != "" {
			expectedParameter = param
		}
	}
	assert.NotEqual(t, "", expectedParameter.Justification)
	assert.NotEqual(t, "", expectedParameter.CostImpact)

	_ = PrettyPrint(result)
}
