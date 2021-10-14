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
	definition, err := ReadPolicyDefinitionFromExcel(
		filepath.Join(base, "Azure Cloud Foundation - Baseline Policies - 20210929.xlsx"),
		[]string{"Management", "Production", "Non-Prod", "Sandbox"},
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
	definition, err := ReadPolicyDefinitionFromExcel(
		filepath.Join(base, "Azure Cloud Foundation - Baseline Policies - 20211014.xlsx"),
		[]string{"ACF's subscriptions", "PLATFORM's subscriptions", "LANDING ZONE's subscriptions", "SANDBOX's subscriptions"},
	)
	assert.NoError(t, err)

	filenameMappings := map[string]string{
		"ACF's subscriptions":          "ASC_policy_ACF.json",
		"PLATFORM's subscriptions":     "ASC_policy_PLATFORM.json",
		"LANDING ZONE's subscriptions": "ASC_policy_LANDINGZONE.json",
		"SANDBOX's subscriptions":      "ASC_policy_SANDBOX.json",
	}
	err = ExportPolicySetParametersAsJSON(definition.ASCPolicySetParameters, filenameMappings, base)
	assert.NoError(t, err)
}
