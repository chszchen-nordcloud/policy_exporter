package exporter

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
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
	config.ExcelFilePath = getTargetFileName(config.TargetDir)

	// A flag to skip the exporting part.
	if !skipExportDuringTest() {
		err = ExportIntermediateFiles(ctx, *config)
		assert.NoError(t, err)
	}

	excelProvider, err := getIntermediateExcelFileProvider(*config)
	assert.NoError(t, err)

	azureAPIProvider, err := getAzureAPIProvider(*config)
	assert.NoError(t, err)

	localRepositoryProvider, err := getLocalLandingZoneRepositoryProvider(*config)
	assert.NoError(t, err)

	var oldExcelProvider PolicyDefinitionProvider
	if config.OldBaselineExcelFilePath != "" {
		provider, err := getOldBaselineExcelFileProvider(*config)
		assert.NoError(t, err)
		oldExcelProvider = *provider
	}

	builtinPoliciesResultReader := newCachedPolicyReader(excelProvider.BuiltInPolicyReader)
	builtinPoliciesSourceReader := newCachedPolicyReader(azureAPIProvider.BuiltInPolicyReader)
	verifyPoliciesFromReaders(t, ctx, builtinPoliciesResultReader, builtinPoliciesSourceReader, oldExcelProvider.BuiltInPolicyReader)
	verifyPolicyRowsFromReader(ctx, t, config, builtinPoliciesSheetReader, builtinPoliciesSourceReader, []policyRowVerifier{
		policyBelongInitiativeVerifier,
		policyEffectVerifier,
	})

	customPoliciesResultReader := newCachedPolicyReader(excelProvider.CustomPolicyReader)
	customPoliciesSourceReader := newCachedPolicyReader(localRepositoryProvider.CustomPolicyReader)
	verifyPoliciesFromReaders(t, ctx, customPoliciesResultReader, customPoliciesSourceReader, oldExcelProvider.CustomPolicyReader)
	verifyPolicyRowsFromReader(ctx, t, config, customPoliciesSheetReader, customPoliciesSourceReader, []policyRowVerifier{
		policyEffectVerifier,
	})

	ascPolicyParametersResultReader := newCachedPolicyParameterReader(excelProvider.ASCPolicySetParameterReader)
	ascPolicyParametersSourceReader := newCachedPolicyParameterReader(azureAPIProvider.ASCPolicySetParameterReader)
	verifyPolicyParametersFromReaders(t, ctx, ascPolicyParametersResultReader, ascPolicyParametersSourceReader, oldExcelProvider.ASCPolicySetParameterReader)
	verifyPolicyParameterRowsFromReader(t, ctx, config, ascPolicyParametersSheetReader, ascPolicyParametersSourceReader)
}

type policyVerifier = func(*testing.T, *Policy, *Policy)
type policyRowVerifier = func(*testing.T, *namedCells, *Policy)
type policyParameterVerifier = func(*testing.T, *PolicyParameter, *PolicyParameter)
type policyParameterRowVerifier = func(*testing.T, *namedCells, *PolicyParameter)

func verifyPolicies(
	t *testing.T, results []Policy, source []Policy, optional bool,
	policyVerifiers []policyVerifier,
) {
	sourcePolicies := make(map[interface{}]*Policy)
	for i := range source {
		k := source[i].ID()
		assert.NotEmpty(t, k)
		sourcePolicies[k] = &source[i]
	}

	for i := range results {
		k := results[i].ID()
		assert.NotEmpty(t, k)
		sourcePolicy, ok := sourcePolicies[k]
		if !optional {
			assert.True(t, ok)
		}

		if sourcePolicy != nil {
			for _, verifier := range policyVerifiers {
				verifier(t, &results[i], sourcePolicy)
			}
		}
	}
}

func verifyPolicyParameters(
	t *testing.T, results []PolicyParameter, source []PolicyParameter, optional bool,
	policyVerifiers []policyParameterVerifier,
) {
	sourceParameters := make(map[interface{}]*PolicyParameter)
	for i := range source {
		k := source[i].ID()
		assert.NotEmpty(t, k)
		sourceParameters[k] = &source[i]
	}

	for i := range results {
		k := results[i].ID()
		assert.NotEmpty(t, k)
		sourceParam, ok := sourceParameters[k]
		if !optional {
			assert.True(t, ok)
		}

		if sourceParam != nil {
			for _, verifier := range policyVerifiers {
				verifier(t, &results[i], sourceParam)
			}
		}
	}
}

func verifyPolicyRows(
	t *testing.T, results []namedCells, source []Policy, optional bool,
	rowVerifiers []policyRowVerifier,
) {
	sourcePolicies := make(map[interface{}]*Policy)
	for i := range source {
		k := source[i].ID()
		assert.NotEmpty(t, k)
		sourcePolicies[source[i].DisplayName] = &source[i]
	}

	for i := range results {
		k := results[i].ID()
		assert.NotEmpty(t, k)
		sourcePolicy, ok := sourcePolicies[k]
		if !optional {
			assert.True(t, ok)
		}

		if sourcePolicy != nil {
			for _, verifier := range rowVerifiers {
				verifier(t, &results[i], sourcePolicy)
			}
		}
	}
}

func verifyPolicyParameterRows(
	t *testing.T, results []namedCells, source []PolicyParameter, optional bool,
	rowVerifiers []policyParameterRowVerifier,
) {
	sourceParameters := make(map[interface{}]*PolicyParameter)
	for i := range source {
		k := source[i].ID()
		assert.NotEmpty(t, k)
		sourceParameters[source[i].ID()] = &source[i]
	}

	for i := range results {
		k := results[i].ID()
		assert.NotEmpty(t, k)
		sourceParam, ok := sourceParameters[k]
		if !optional {
			assert.True(t, ok)
		}

		if sourceParam != nil {
			for _, verifier := range rowVerifiers {
				verifier(t, &results[i], sourceParam)
			}
		}
	}
}

func policyEffectVerifier(t *testing.T, row *namedCells, source *Policy) {
	defaultValuesStr, ok := row.Get(ColumnDefaultValues)
	assert.True(t, ok)
	possibleValuesStr, ok := row.Get(ColumnPossibleValues)
	assert.True(t, ok)
	defaultValues := parsePolicyParameterRawValues(defaultValuesStr)
	possibleValues := parsePolicyParameterRawValues(possibleValuesStr)
	defaultEffect, ok := defaultValues["effect"]
	if source.IsInitiative {
		assert.False(t, ok)
	} else {
		if ok {
			possibleEffects, ok := possibleValues["effect"]
			assert.True(t, ok)
			assert.NotEmpty(t, defaultEffect)
			assert.NotEmpty(t, possibleEffects)
		} else {
			_, ok := defaultValues["*effect"]
			assert.True(t, ok)
			_, ok = possibleValues["*effect"]
			assert.False(t, ok)
		}
	}
}

func policyBelongInitiativeVerifier(t *testing.T, row *namedCells, source *Policy) {
	v, ok := row.Get(ColumnBelongingInitiatives)
	assert.True(t, ok)
	if len(source.InitiativeIDs) > 0 {
		assert.NotEmpty(t, v)
	}
}

func verifyPolicyRowsFromReader(ctx context.Context, t *testing.T, config *Config, policySheetReader policySheetReader, baseReader PolicyReader, verifiers []policyRowVerifier) {
	f, err := excelize.OpenFile(getTargetFileName(config.TargetDir))
	assert.NoError(t, err)
	rows, err := policySheetReader.readRows(f, config.ManagementGroups)
	assert.NoError(t, err)

	policiesFromBase, err := baseReader(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(policiesFromBase), len(rows))

	verifyPolicyRows(t, rows, policiesFromBase, false, verifiers)
}

func verifyPolicyParameterRowsFromReader(
	t *testing.T, ctx context.Context, config *Config,
	sheetReader policyParameterSheetReader,
	baseReader PolicyParameterReader,
) {
	f, err := excelize.OpenFile(getTargetFileName(config.TargetDir))
	assert.NoError(t, err)
	rows, err := sheetReader.readRows(f, config.ManagementGroups)
	assert.NoError(t, err)

	paramsFromBase, err := baseReader(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(paramsFromBase), len(rows))

	verifyPolicyParameterRows(t, rows, paramsFromBase, false, []policyParameterRowVerifier{
		// Put any ASC policy parameter verifiers here
	})
}

func verifyPoliciesFromReaders(t *testing.T, ctx context.Context, resultPolicyReader PolicyReader, baseReader PolicyReader, oldExcelReader PolicyReader) {
	resultPolicies, err := resultPolicyReader(ctx)
	assert.NoError(t, err)
	policiesFromBase, err := baseReader(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(policiesFromBase), len(resultPolicies))

	verifyPolicies(t, resultPolicies, policiesFromBase, false, []policyVerifier{})

	// Verify that 'justification' and 'cost impact' of policies from old baseline file are kept.
	if oldExcelReader != nil {
		builtinPoliciesFromOldExcel, err := oldExcelReader(ctx)
		assert.NoError(t, err)

		verifyPolicies(t, resultPolicies, builtinPoliciesFromOldExcel, true,
			[]policyVerifier{
				func(t *testing.T, policy *Policy, source *Policy) {
					if source.Justification != "" {
						assert.Equal(t, policy.Justification, source.Justification)

					}
					if source.CostImpact != "" {
						assert.Equal(t, policy.CostImpact, source.CostImpact)
					}
				},
			},
		)
	}
}

// Support flexible handlings instead of JSON formatting for arrays,
// Add a column to the end for recommendation, whose value is inferred from the current baseline.
func verifyPolicyParametersFromReaders(t *testing.T, ctx context.Context, resultReader PolicyParameterReader, baseReader PolicyParameterReader, oldExcelReader PolicyParameterReader) {
	resultParameters, err := resultReader(ctx)
	assert.NoError(t, err)
	parametersFromBase, err := baseReader(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(parametersFromBase), len(resultParameters))

	verifyPolicyParameters(t, resultParameters, parametersFromBase, false, []policyParameterVerifier{
		// Put any policy parameter verifiers here
	})

	// Verify that 'justification' and 'cost impact' of policies from old baseline file are kept.
	if oldExcelReader != nil {
		builtinPoliciesFromOldExcel, err := oldExcelReader(ctx)
		assert.NoError(t, err)

		verifyPolicyParameters(t, resultParameters, builtinPoliciesFromOldExcel, true,
			[]policyParameterVerifier{
				func(t *testing.T, policy *PolicyParameter, source *PolicyParameter) {
					if source.Justification != "" {
						assert.Equal(t, policy.Justification, source.Justification)

					}
					if source.CostImpact != "" {
						assert.Equal(t, policy.CostImpact, source.CostImpact)
					}
				},
			},
		)
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

	err = config.Validate()
	assert.NoError(t, err)

	return config
}

func skipExportDuringTest() bool {
	return os.Getenv("SKIP_EXPORT") == "true"
}

func newCachedPolicyParameterReader(wrapped PolicyParameterReader) PolicyParameterReader {
	var cached []PolicyParameter
	return func(ctx context.Context) ([]PolicyParameter, error) {
		if cached == nil {
			parameters, err := wrapped(ctx)
			if err != nil {
				return nil, err
			}
			cached = parameters
		}
		return cached, nil
	}
}

func newCachedPolicyReader(wrapped PolicyReader) PolicyReader {
	var cached []Policy
	return func(ctx context.Context) ([]Policy, error) {
		if cached == nil {
			policies, err := wrapped(ctx)
			if err != nil {
				return nil, err
			}
			cached = policies
		}
		return cached, nil
	}
}
