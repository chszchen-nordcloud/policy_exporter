package exporter

import (
	"github.com/stretchr/testify/assert"
	"os"
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
	localLandingZoneRepoDir := os.Getenv("LOCAL_LZ_REPO_DIR")
	err = ExportPoliciesAsJSON(policies, localLandingZoneRepoDir, base)
	assert.NoError(t, err)
}

func TestExportPolicySetParametersAsJSON(t *testing.T) {
	if SkipTest() {
		return
	}
	base := TestResourceDir()
	definition, err := ReadPolicyDefinitionFromExcel(
		filepath.Join(base, "Azure Cloud Foundation - Baseline Policies - 20210929.xlsx"),
		[]string{"Management", "Production", "Non-Prod", "Sandbox"},
	)
	assert.NoError(t, err)

	err = ExportPolicySetParametersAsJSON(definition.ASCPolicySetParameters, "Prod", base)
	assert.NoError(t, err)
}
