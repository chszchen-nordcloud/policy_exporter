package exporter

import "context"

func SyncAndExport(ctx context.Context, config Config) error {
	if err := config.Validate(); err != nil {
		return err
	}

	providers, err := getPolicyDefinitionProviders(config)
	if err != nil {
		return err
	}

	var builtinPolicyByName map[string]Policy
	var customPolicyByName map[string]Policy
	var ascPolicySetParameterByName map[string]PolicyParameter
	for _, provider := range providers {
		builtinPolicies, err := (provider.BuiltInPolicyReader)(ctx)
		if err != nil {
			return err
		}
		mergePolicies(builtinPolicies, &builtinPolicyByName)

		customPolicies, err := (provider.CustomPolicyReader)(ctx)
		if err != nil {
			return err
		}
		mergePolicies(customPolicies, &customPolicyByName)

		params, err := (provider.ASCPolicySetParameterReader)(ctx)
		if err != nil {
			return err
		}
		mergePolicySetParameters(params, &ascPolicySetParameterByName)
	}

	builtinPolicies := collectPolicies(builtinPolicyByName)
	customPolicies := collectPolicies(customPolicyByName)
	policySetParams := collectPolicySetParameters(ascPolicySetParameterByName)
	var policies []Policy
	policies = append(policies, builtinPolicies...)
	policies = append(policies, customPolicies...)

	for _, exporter := range getPolicyDefinitionExporters(config) {
		if exporter.PolicyExporter != nil {
			if err := (exporter.PolicyExporter)(policies, config.TargetDir); err != nil {
				return err
			}
		} else {
			if exporter.BuiltInPolicyExporter != nil {
				if err := (exporter.BuiltInPolicyExporter)(policies, config.TargetDir); err != nil {
					return err
				}
			}
			if exporter.CustomPolicyExporter != nil {
				if err := (exporter.CustomPolicyExporter)(policies, config.TargetDir); err != nil {
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
		if existingPolicy, ok := (*dest)[k]; ok {
			existingPolicy.Merge(policy)
			(*dest)[k] = existingPolicy
		} else {
			(*dest)[k] = policy
		}
	}
}

func mergePolicySetParameters(params []PolicyParameter, dest *map[string]PolicyParameter) {
	for _, param := range params {
		k := param.DisplayName
		if existingParam, ok := (*dest)[k]; ok {
			existingParam.Merge(param)
			(*dest)[k] = existingParam
		} else {
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
			return ExportPolicySetParametersAsJSON(params, "Prod", targetDir)
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

func getPolicyDefinitionProviders(config Config) ([]*PolicyDefinitionProvider, error) {
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

	excelPolicyDef, err := ReadPolicyDefinitionFromExcel(config.ExcelFilePath, SHEET_NAME_BUILTIN_POLICIES, SHEET_NAME_CUSTOM_POLICIES, SHEET_NAME_ASC_PARAMETERS)
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
		excelProvider,
		yamlProvider,
	}, nil
}
