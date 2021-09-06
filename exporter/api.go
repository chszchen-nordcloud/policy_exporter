package exporter

import "context"

func SyncAndExport(ctx context.Context, config Config) error {
	providers, err := getPolicyDefinitionProviders(config)
	if err != nil {
		return err
	}

	var policyByName map[string]Policy
	var policySetParameterByName map[string]PolicySetParameter
	for _, provider := range providers {
		policies, err := (provider.PolicyReader)(ctx)
		if err != nil {
			return err
		}
		mergePolicies(policies, &policyByName)

		params, err := (provider.PolicySetParameterReader)(ctx)
		if err != nil {
			return err
		}
		mergePolicySetParameters(params, &policySetParameterByName)
	}

	policies := collectPolicies(policyByName)
	policySetParams := collectPolicySetParameters(policySetParameterByName)
	for _, exporter := range getPolicyDefinitionExporters() {
		if err := (exporter.PolicyExporter)(ctx, policies, config.LandingZoneRepoLocalPath); err != nil {
			return err
		}
		if err := (exporter.PolicySetParameterExporter)(ctx, policySetParams, config.LandingZoneRepoLocalPath); err != nil {
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

func mergePolicySetParameters(params []PolicySetParameter, dest *map[string]PolicySetParameter) {
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

func collectPolicySetParameters(paramByName map[string]PolicySetParameter) []PolicySetParameter {
	result := make([]PolicySetParameter, 0, len(paramByName))
	for _, v := range paramByName {
		result = append(result, v)
	}
	return result
}

func getPolicyDefinitionExporters() []PolicyDefinitionExporter {
	return []PolicyDefinitionExporter{
		PolicyDefinitionExporter{
			PolicyExporter:             ExportPoliciesAsJson,
			PolicySetParameterExporter: ExportPolicySetParametersAsJson,
		},
	}
}

func getPolicyDefinitionProviders(config Config) ([]*PolicyDefinitionProvider, error) {
	azureAPIProvider, err := NewAzureAPIPolicyDefinitionProvider(config)
	if err != nil {
		return nil, err
	}

	// excel provider

	return []*PolicyDefinitionProvider{azureAPIProvider}, nil
}
