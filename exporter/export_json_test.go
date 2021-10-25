package exporter

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestExportPoliciesAsJson(t *testing.T) {
	if SkipTest() {
		return
	}
	base := TestResourceDir()
	config := getConfigForTest(t)
	definition, err := ReadPolicyDefinitionFromExcel(
		filepath.Join(base, "Azure Cloud Foundation - Baseline Policies - 20210929.xlsx"),
		config.ManagementGroups,
		config.Subscriptions,
	)
	assert.NoError(t, err)

	var policies []Policy
	policies = append(policies, definition.BuiltInPolicies...)
	policies = append(policies, definition.CustomPolicies...)
	err = ExportPoliciesAsJSON(policies, base)
	assert.NoError(t, err)
}

func TestExportPolicySetParametersAsJSON(t *testing.T) {
	if SkipTest() {
		return
	}
	base := TestResourceDir()
	config := getConfigForTest(t)
	definition, err := ReadPolicyDefinitionFromExcel(
		filepath.Join(base, "Azure Cloud Foundation - Baseline Policies - 20211014.xlsx"),
		config.ManagementGroups,
		config.Subscriptions,
	)
	assert.NoError(t, err)

	err = ExportPolicySetParametersAsJSON(definition.ASCPolicySetParameters, config.Subscriptions, base)
	assert.NoError(t, err)
}
