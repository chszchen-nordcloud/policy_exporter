package exporter

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestExportIntermediateFiles(t *testing.T) {
	if SkipTest() {
		return
	}

	// There are relative paths in the configuration file which are relative to the parent directory.
	err := os.Chdir("..")
	assert.NoError(t, err)

	ctx := context.Background()
	config := getConfigForAPITest(t)

	// A flag to skip the exporting part.
	if !skipExportDuringTest() {
		err = ExportIntermediateFiles(ctx, *config)
		assert.NoError(t, err)
	}

	excelProvider, err := getIntermediateExcelFileProvider(*config)
	assert.NoError(t, err)

	azureAPIProvider, err := getAzureAPIProvider(*config)
	assert.NoError(t, err)

	builtinPoliciesFromExcel, err := excelProvider.BuiltInPolicyReader(ctx)
	assert.NoError(t, err)

	builtinPoliciesFromAPI, err := azureAPIProvider.BuiltInPolicyReader(ctx)
	assert.NoError(t, err)

	// Verify that all policies from Azure API are in the intermediate Excel file
	assert.Equal(t, len(builtinPoliciesFromAPI), len(builtinPoliciesFromExcel))

	ascParametersFromAPI, err := azureAPIProvider.ASCPolicySetParameterReader(ctx)
	assert.NoError(t, err)

	ascParametersFromExcel, err := excelProvider.ASCPolicySetParameterReader(ctx)
	assert.NoError(t, err)

	// Verify that all ASC policy parameters from Azure API are in the intermediate Excel file
	assert.Equal(t, len(ascParametersFromAPI), len(ascParametersFromExcel))

	localRepositoryProvider, err := getLocalLandingZoneRepositoryProvider(*config)
	assert.NoError(t, err)

	customPoliciesFromExcel, err := excelProvider.CustomPolicyReader(ctx)
	assert.NoError(t, err)

	customPoliciesFromLocalRepo, err := localRepositoryProvider.CustomPolicyReader(ctx)
	assert.NoError(t, err)

	// Verify that all policies from local repository are in the intermediate Excel file
	assert.Equal(t, len(customPoliciesFromLocalRepo), len(customPoliciesFromExcel))

	if config.OldBaselineExcelFilePath != "" {
		oldExcelProvider, err := getOldBaselineExcelFileProvider(*config)
		assert.NoError(t, err)

		builtinPoliciesFromOldExcel, err := oldExcelProvider.BuiltInPolicyReader(ctx)
		assert.NoError(t, err)

		matchPolicyWhen := func(p Policy) bool { return p.Justification != "" || p.CostImpact != "" }
		matchPolicy := func(p1 Policy, p2 Policy) bool { return p1.DisplayName == p2.DisplayName }

		// Verify that 'justification' and 'cost impact' of policies from old baseline file are kept.
		assertPolicyMatchesWhen(
			t, builtinPoliciesFromOldExcel, builtinPoliciesFromExcel, matchPolicyWhen, matchPolicy,
			func(expected Policy, value Policy) {
				assert.Equal(t, expected.Justification, value.Justification)
				assert.Equal(t, expected.CostImpact, value.CostImpact)
			},
		)

		matchPolicyParameterWhen := func(p PolicyParameter) bool { return p.Justification != "" || p.CostImpact != "" }
		matchPolicyParameter := func(p1 PolicyParameter, p2 PolicyParameter) bool { return p1.InternalName == p2.InternalName }

		// Verify that 'justification' and 'cost impact' of ASC policy parameters from old baseline file are kept.
		ascParametersFromOldExcel, err := oldExcelProvider.ASCPolicySetParameterReader(ctx)
		assert.NoError(t, err)
		assertPolicyParameterMatchesWhen(
			t, ascParametersFromOldExcel, ascParametersFromExcel, matchPolicyParameterWhen, matchPolicyParameter,
			func(expected PolicyParameter, value PolicyParameter) {
				assert.Equal(t, expected.Justification, value.Justification)
				assert.Equal(t, expected.CostImpact, value.CostImpact)
			},
		)
	}

}

func assertPolicyMatchesWhen(t *testing.T, values []Policy, expectedValues []Policy,
	when func(Policy) bool, match func(Policy, Policy) bool,
	verify func(Policy, Policy)) {
	for _, expected := range expectedValues {
		var matched *Policy
		if when(expected) {
			for i := range values {
				if match(expected, values[i]) {
					matched = &values[i]
					break
				}
			}
			if matched != nil {
				verify(expected, *matched)
			}
		}
	}
}

func TestExportFinal(t *testing.T) {
	if SkipTest() {
		return
	}

	// There are relative paths in the configuration file which are relative to the parent directory.
	err := os.Chdir("..")
	assert.NoError(t, err)

	ctx := context.Background()
	config := getConfigForAPITest(t)

	// A flag to skip the exporting part.
	if !skipExportDuringTest() {
		err := ExportFinal(ctx, *config)
		assert.NoError(t, err)
	}
}

func getConfigForAPITest(t *testing.T) *Config {
	resourceDir := "test_resources"
	configFilePath := filepath.Join(resourceDir, "example_config.yaml")
	config, err := buildConfig(&configFilePath, []string{resourceDir})
	assert.NoError(t, err)
	config.ExcelFilePath = getTargetFileName(resourceDir)

	err = config.Validate()
	assert.NoError(t, err)

	return config
}

func assertPolicyParameterMatchesWhen(t *testing.T, values []PolicyParameter, expectedValues []PolicyParameter,
	when func(PolicyParameter) bool, match func(PolicyParameter, PolicyParameter) bool,
	verify func(PolicyParameter, PolicyParameter)) {
	for _, expected := range expectedValues {
		var matched *PolicyParameter
		if when(expected) {
			for i := range values {
				if match(expected, values[i]) {
					matched = &values[i]
					break
				}
			}
			if matched != nil {
				verify(expected, *matched)
			}
		}
	}
}

func skipExportDuringTest() bool {
	return os.Getenv("SKIP_EXPORT") == "true"
}
