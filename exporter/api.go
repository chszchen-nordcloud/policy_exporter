package exporter

import (
	"context"
	"fmt"
	"github.com/xuri/excelize/v2"
	"sort"
)

func ExportIntermediateFiles(ctx context.Context, config Config) error {
	if err := config.Validate(); err != nil {
		return err
	}
	providers, err := getPolicyDefinitionProvidersForIntermediateExport(config)
	if err != nil {
		return err
	}
	return syncAndExport(ctx, config, providers, getIntermediateExporters(config))
}

func ExportFinal(ctx context.Context, config Config) error {
	if err := config.Validate(); err != nil {
		return err
	}
	providers, err := getPolicyDefinitionProvidersForFinalExport(config)
	if err != nil {
		return err
	}
	return syncAndExport(ctx, config, providers, getPolicyDefinitionExporters(config))
}

// syncAndExport syncs content between providers and exported files. 'providers' provides values of the same type in order.
// The values are of type UniqueResource and will be merged and sorted before exported. Sorting is added so that it is
// easy to tell whether the content of the exported files have changed for humans.
func syncAndExport(ctx context.Context, config Config, providers []*PolicyDefinitionProvider, exporters []PolicyDefinitionExporter) error {
	builtinPolicyByName := make(map[string]Policy)
	customPolicyByName := make(map[string]Policy)
	ascPolicySetParameterByName := make(map[string]PolicyParameter)
	for _, provider := range providers {
		if provider.BuiltInPolicyReader != nil {
			builtinPolicies, err := (provider.BuiltInPolicyReader)(ctx)
			if err != nil {
				return err
			}
			mergePolicies(builtinPolicies, &builtinPolicyByName)
		}

		if provider.CustomPolicyReader != nil {
			customPolicies, err := (provider.CustomPolicyReader)(ctx)
			if err != nil {
				return err
			}
			mergePolicies(customPolicies, &customPolicyByName)
		}

		if provider.ASCPolicySetParameterReader != nil {
			params, err := (provider.ASCPolicySetParameterReader)(ctx)
			if err != nil {
				return err
			}
			mergePolicySetParameters(params, &ascPolicySetParameterByName)
		}
	}

	builtinPolicies := collectPolicies(builtinPolicyByName)
	customPolicies := collectPolicies(customPolicyByName)
	policySetParams := collectPolicySetParameters(ascPolicySetParameterByName)
	var policies []Policy
	policies = append(policies, builtinPolicies...)
	policies = append(policies, customPolicies...)

	sort.Sort(SortPoliciesByDisplayName(builtinPolicies))
	sort.Sort(SortPoliciesByDisplayName(customPolicies))
	sort.Sort(SortPoliciesByDisplayName(policies))
	sort.Sort(SortPolicyParametersByInternalName(policySetParams))

	for _, exporter := range exporters {
		if exporter.PolicyExporter != nil {
			if err := (exporter.PolicyExporter)(policies, config.TargetDir); err != nil {
				return err
			}
		} else {
			if exporter.BuiltInPolicyExporter != nil {
				if err := (exporter.BuiltInPolicyExporter)(builtinPolicies, config.TargetDir); err != nil {
					return err
				}
			}
			if exporter.CustomPolicyExporter != nil {
				if err := (exporter.CustomPolicyExporter)(customPolicies, config.TargetDir); err != nil {
					return err
				}
			}
		}

		if err := (exporter.PolicySetParameterExporter)(policySetParams, config.TargetDir); err != nil {
			return err
		}
	}

	return nil
}

// mergePolicies merges a set of policies into destination map.
func mergePolicies(policies []Policy, dest *map[string]Policy) {
	initialize := false
	if len(*dest) == 0 {
		initialize = true
	}
	for _, policy := range policies {
		k := policy.DisplayName
		if k == "" {
			fmt.Printf("ingore policy whose name is empty: %v\n", policy)
			continue
		}
		if initialize {
			(*dest)[k] = policy
		} else if existingPolicy, ok := (*dest)[k]; ok {
			(&existingPolicy).Merge(&policy) //nolint:gosec It's safe here as we don't keep address.
			(*dest)[k] = existingPolicy
		}
	}
}

func mergePolicySetParameters(params []PolicyParameter, dest *map[string]PolicyParameter) {
	initialize := false
	if len(*dest) == 0 {
		initialize = true
	}
	for _, param := range params {
		k := param.InternalName
		if k == "" {
			fmt.Printf("ingore parameter whose name is empty: %v\n", param)
			continue
		}
		if initialize {
			(*dest)[k] = param
		} else if existingParam, ok := (*dest)[k]; ok {
			(&existingParam).Merge(&param)
			(*dest)[k] = existingParam
		}
	}
}

// collectPolicies collects values from a map.
func collectPolicies(policyByName map[string]Policy) []Policy {
	result := make([]Policy, 0, len(policyByName))
	for _, v := range policyByName {
		result = append(result, v)
	}
	return result
}

func collectPolicySetParameters(paramByName map[string]PolicyParameter) []PolicyParameter {
	result := make([]PolicyParameter, 0, len(paramByName))
	for _, v := range paramByName {
		result = append(result, v)
	}
	return result
}

// getPolicyDefinitionExporters returns all final exporters.
func getPolicyDefinitionExporters(config Config) []PolicyDefinitionExporter {
	jsonExporter := PolicyDefinitionExporter{
		PolicyExporter: ExportPoliciesAsJSON,
		PolicySetParameterExporter: func(params []PolicyParameter, targetDir string) error {
			return ExportPolicySetParametersAsJSON(params, config.Subscriptions, targetDir)
		},
	}
	mdxExport := PolicyDefinitionExporter{
		BuiltInPolicyExporter: func(policies []Policy, targetDir string) error {
			return ExportPoliciesDoc("BuiltInPolicies.mdx", config.ManagementGroups, policies, targetDir)
		},
		CustomPolicyExporter: func(policies []Policy, targetDir string) error {
			return ExportPoliciesDoc("CustomPolicies.mdx", config.ManagementGroups, policies, targetDir)
		},
		PolicySetParameterExporter: func(params []PolicyParameter, targetDir string) error {
			return ExportASCPolicyParametersDoc(config.Subscriptions, params, targetDir)
		},
	}
	return []PolicyDefinitionExporter{
		jsonExporter, mdxExport,
	}
}

// getIntermediateExporters returns exporters for intermediate exporting, currently only excel exporter is included.
func getIntermediateExporters(config Config) []PolicyDefinitionExporter {
	f := excelize.NewFile()
	excelExporter := PolicyDefinitionExporter{
		BuiltInPolicyExporter: func(policies []Policy, targetDir string) error {
			err := ExportDataToExcelSheet(f, SheetBuiltInPolicy, policies, config.ManagementGroups)
			if err != nil {
				return err
			}
			return SaveExcelFile(f, targetDir)
		},
		CustomPolicyExporter: func(policies []Policy, targetDir string) error {
			err := ExportDataToExcelSheet(f, SheetCustomPolicy, policies, config.ManagementGroups)
			if err != nil {
				return err
			}
			return SaveExcelFile(f, targetDir)
		},
		PolicySetParameterExporter: func(parameters []PolicyParameter, targetDir string) error {
			err := ExportDataToExcelSheet(f, SheetASCParameters, parameters, config.Subscriptions)
			if err != nil {
				return err
			}
			return SaveExcelFile(f, targetDir)
		},
	}
	return []PolicyDefinitionExporter{excelExporter}
}

func getPolicyDefinitionProvidersForFinalExport(config Config) ([]*PolicyDefinitionProvider, error) {
	intermediateExporters, err := getPolicyDefinitionProvidersForIntermediateExport(config)
	if err != nil {
		return nil, err
	}

	intermediateExcelFileProvider, err := getIntermediateExcelFileProvider(config.ExcelFilePath, config)
	if err != nil {
		return nil, err
	}

	result := make([]*PolicyDefinitionProvider, 0, len(intermediateExporters)+1)
	result = append(result, intermediateExporters...)
	result = append(result, intermediateExcelFileProvider)
	return result, nil
}

func getPolicyDefinitionProvidersForIntermediateExport(config Config) ([]*PolicyDefinitionProvider, error) {
	azureAPIProvider, err := getAzureAPIProvider(config)
	if err != nil {
		return nil, err
	}

	localRepoProvider, err := getLocalLandingZoneRepositoryProvider(config)
	if err != nil {
		return nil, err
	}

	result := []*PolicyDefinitionProvider{
		azureAPIProvider,
		localRepoProvider,
	}

	if config.OldBaselineExcelFilePath != "" {
		oldExcelProvider, err := getOldBaselineExcelFileProvider(config)
		if err != nil {
			return nil, err
		}
		result = append(result, oldExcelProvider)
	}

	//if config.YAMLFilePath != "" {
	//	yamlProvider, err := getYAMLFileProvider(config)
	//	if err != nil {
	//		return nil, err
	//	}
	//	result = append(result, yamlProvider)
	//}

	return result, nil
}

func getIntermediateExcelFileProvider(path string, config Config) (*PolicyDefinitionProvider, error) {
	excelPolicyDef, err := ReadPolicyDefinitionFromExcel(path, config.ManagementGroups, config.Subscriptions)
	if err != nil {
		return nil, err
	}
	excelProvider := &PolicyDefinitionProvider{
		BuiltInPolicyReader: func(ctx context.Context) ([]Policy, error) {
			return excelPolicyDef.BuiltInPolicies, nil
		},
		CustomPolicyReader: func(ctx context.Context) ([]Policy, error) {
			return excelPolicyDef.CustomPolicies, nil
		},
		ASCPolicySetParameterReader: func(ctx context.Context) ([]PolicyParameter, error) {
			return excelPolicyDef.ASCPolicySetParameters, nil
		},
	}
	return excelProvider, nil
}

func getAzureAPIProvider(config Config) (*PolicyDefinitionProvider, error) {
	api, err := NewAzureAPI(config.SubscriptionID)
	if err != nil {
		return nil, err
	}
	azureAPIProvider := &PolicyDefinitionProvider{
		BuiltInPolicyReader: func(ctx context.Context) ([]Policy, error) {
			return api.ListBuiltInPolicyByManagementGroup(ctx, config.PolicyQueryManagementGroupName)
		},
		ASCPolicySetParameterReader: func(ctx context.Context) ([]PolicyParameter, error) {
			return api.GetPolicySetParameters(ctx, config.PolicyQueryASCPolicySetName)
		},
	}
	return azureAPIProvider, nil
}

func getLocalLandingZoneRepositoryProvider(config Config) (*PolicyDefinitionProvider, error) {
	localRepoDef, err := ReadCustomPoliciesFromLocalRepository(config.LocalLandingZoneRepoDir)
	if err != nil {
		return nil, err
	}
	localRepoProvider := &PolicyDefinitionProvider{
		CustomPolicyReader: func(ctx context.Context) ([]Policy, error) {
			return localRepoDef, nil
		},
	}
	return localRepoProvider, nil
}

func getOldBaselineExcelFileProvider(config Config) (*PolicyDefinitionProvider, error) {
	return getIntermediateExcelFileProvider(config.OldBaselineExcelFilePath, config)
}
