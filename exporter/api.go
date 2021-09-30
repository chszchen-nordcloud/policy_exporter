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

func mergePolicies(policies []Policy, dest *map[string]Policy) {
	for _, policy := range policies {
		k := policy.DisplayName
		if k == "" {
			fmt.Printf("ingore policy whose name is empty: %v\n", policy)
			continue
		}
		if existingPolicy, ok := (*dest)[k]; ok {
			existingPolicy.Merge(policy)
			(*dest)[k] = existingPolicy
		} else if policy.AlwaysIncludedInExport {
			(*dest)[k] = policy
		}
	}
}

func mergePolicySetParameters(params []PolicyParameter, dest *map[string]PolicyParameter) {
	for _, param := range params {
		k := param.InternalName
		if k == "" {
			fmt.Printf("ingore parameter whose name is empty: %v\n", param)
			continue
		}
		if existingParam, ok := (*dest)[k]; ok {
			existingParam.Merge(param)
			(*dest)[k] = existingParam
		} else if param.AlwaysIncludedInExport {
			(*dest)[k] = param
		}
	}
}

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

func getPolicyDefinitionExporters(config Config) []PolicyDefinitionExporter {
	jsonExporter := PolicyDefinitionExporter{
		PolicyExporter: func(policies []Policy, targetDir string) error {
			return ExportPoliciesAsJSON(policies, config.LocalLandingZoneRepoDir, targetDir)
		},
		PolicySetParameterExporter: func(params []PolicyParameter, targetDir string) error {
			return ExportPolicySetParametersAsJSON(params, config.ManagementGroupToExportForASCParameters, targetDir)
		},
	}
	mdxExport := PolicyDefinitionExporter{
		BuiltInPolicyExporter: func(policies []Policy, targetDir string) error {
			return ExportBuiltInPolicyDoc(config.ManagementGroups, policies, targetDir)
		},
		PolicySetParameterExporter: func(params []PolicyParameter, targetDir string) error {
			return ExportASCPolicyDoc(config.ManagementGroups, params, targetDir)
		},
	}
	return []PolicyDefinitionExporter{
		jsonExporter, mdxExport,
	}
}

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
			err := ExportDataToExcelSheet(f, SheetASCParameters, parameters, config.ManagementGroups)
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

	excelPolicyDef, err := ReadPolicyDefinitionFromExcel(config.ExcelFilePath, config.ManagementGroups)
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

	result := make([]*PolicyDefinitionProvider, 0, len(intermediateExporters)+1)
	result = append(result, intermediateExporters...)
	result = append(result, excelProvider)
	return result, nil
}

func getPolicyDefinitionProvidersForIntermediateExport(config Config) ([]*PolicyDefinitionProvider, error) {
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

	localRepoDef, err := ReadCustomPoliciesFromLocalRepository(config.LocalLandingZoneRepoDir)
	if err != nil {
		return nil, err
	}
	localRepoProvider := &PolicyDefinitionProvider{
		CustomPolicyReader: func(ctx context.Context) ([]Policy, error) {
			return localRepoDef, nil
		},
	}

	excelPolicyDef, err := ReadPolicyDefinitionFromObsoleteExcel(config.OldBaselineExcelFilePath, config.ManagementGroups)
	if err != nil {
		return nil, err
	}
	excelProvider := &PolicyDefinitionProvider{
		BuiltInPolicyReader: func(ctx context.Context) ([]Policy, error) {
			return excelPolicyDef.BuiltInPolicies, nil
		},
		ASCPolicySetParameterReader: func(ctx context.Context) ([]PolicyParameter, error) {
			return excelPolicyDef.ASCPolicySetParameters, nil
		},
	}

	yamlPolicyDef, err := ReadPolicyDefinitionFromYAML(config.YAMLFilePath)
	if err != nil {
		return nil, err
	}
	yamlProvider := &PolicyDefinitionProvider{
		BuiltInPolicyReader: func(ctx context.Context) ([]Policy, error) {
			return yamlPolicyDef.BuiltInPolicies, nil
		},
		CustomPolicyReader: func(ctx context.Context) ([]Policy, error) {
			return yamlPolicyDef.CustomPolicies, nil
		},
		ASCPolicySetParameterReader: func(ctx context.Context) ([]PolicyParameter, error) {
			return yamlPolicyDef.ASCPolicySetParameters, nil
		},
	}

	return []*PolicyDefinitionProvider{
		azureAPIProvider,
		localRepoProvider,
		excelProvider,
		yamlProvider,
	}, nil
}
