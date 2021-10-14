package exporter

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/policy"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"strings"
)

type AzureAPI struct {
	authorizer     autorest.Authorizer
	subscriptionID string
	policySetAPI   policy.SetDefinitionsClient
	policyAPI      policy.DefinitionsClient
}

func NewAzureAPI(subscriptionID string) (*AzureAPI, error) {
	var azureAPI AzureAPI
	azureAPI.subscriptionID = subscriptionID

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}
	azureAPI.authorizer = authorizer

	policySetAPI := policy.NewSetDefinitionsClient(subscriptionID)
	policySetAPI.Authorizer = azureAPI.authorizer
	azureAPI.policySetAPI = policySetAPI

	policyAPI := policy.NewDefinitionsClient(subscriptionID)
	policyAPI.Authorizer = azureAPI.authorizer
	azureAPI.policyAPI = policyAPI

	return &azureAPI, nil
}

// GetPolicySetParameters returns all parameters of a policy set.
func (az *AzureAPI) GetPolicySetParameters(ctx context.Context, policySetName string) ([]PolicyParameter, error) {
	policySet, err := az.policySetAPI.GetBuiltIn(ctx, policySetName)
	if err != nil {
		return nil, err
	}
	result := parsePolicyParameter(policySet.Parameters)
	return result, nil
}

func parsePolicyParameter(paramDefs map[string]*policy.ParameterDefinitionsValue) []PolicyParameter {
	result := make([]PolicyParameter, 0, len(paramDefs))
	for internalName, paramDef := range paramDefs {
		p := PolicyParameter{
			InternalName: internalName,
			Type:         string(paramDef.Type),
			DefaultValue: paramDef.DefaultValue,
		}
		if paramDef.AllowedValues != nil {
			p.AllowedValues = *paramDef.AllowedValues
		}
		if paramDef.Metadata != nil {
			if paramDef.Metadata.Description != nil {
				p.Description = *paramDef.Metadata.Description
			}
			if paramDef.Metadata.DisplayName != nil {
				p.DisplayName = *paramDef.Metadata.DisplayName
			}
		}
		result = append(result, p)
	}
	return result
}

// ListBuiltInPolicyByManagementGroup returns all builtin policies attached to a management group.
func (az *AzureAPI) ListBuiltInPolicyByManagementGroup(ctx context.Context, managementGroupID string) ([]Policy, error) {
	policyPage, err := az.policyAPI.ListByManagementGroup(ctx, managementGroupID)
	if err != nil {
		return nil, err
	}

	policySetPage, err := az.policySetAPI.ListByManagementGroup(ctx, managementGroupID)
	if err != nil {
		return nil, err
	}

	isPreviewOrDeprecated := func(displayName string) bool {
		return strings.HasPrefix(displayName, "[Deprecated]") || strings.HasPrefix(displayName, "[Preview]")
	}

	policyToInitiatives := make(map[string]*[]string)
	var result []Policy
	for list := policy.NewSetDefinitionListResultIterator(policySetPage); list.NotDone(); err = list.NextWithContext(ctx) {
		if err != nil {
			return nil, err
		}
		policySetDef := list.Value()

		displayName := *policySetDef.DisplayName
		if isPreviewOrDeprecated(displayName) {
			continue
		}

		for _, child := range *policySetDef.PolicyDefinitions {
			initiatives, ok := policyToInitiatives[*child.PolicyDefinitionID]
			if !ok {
				initiatives = new([]string)
				policyToInitiatives[*child.PolicyDefinitionID] = initiatives
			}
			*initiatives = append(*initiatives, *policySetDef.ID)
		}

		p, err := policySetDefToPolicy(policySetDef)
		if err != nil {
			return nil, err
		}
		result = append(result, *p)
	}

	for list := policy.NewDefinitionListResultIterator(policyPage); list.NotDone(); err = list.NextWithContext(ctx) {
		if err != nil {
			return nil, err
		}
		policyDef := list.Value()

		displayName := *policyDef.DisplayName
		if isPreviewOrDeprecated(displayName) {
			continue
		}

		p, err := policyDefToPolicy(policyDef)
		if err != nil {
			return nil, err
		}

		if initiativeIDs, ok := policyToInitiatives[*policyDef.ID]; ok {
			p.InitiativeIDs = *initiativeIDs
		}

		result = append(result, *p)
	}
	return result, nil
}

func policySetDefToPolicy(policyDef policy.SetDefinition) (*Policy, error) {
	p := Policy{
		DisplayName:  *policyDef.DisplayName,
		ResourceID:   *policyDef.ID,
		Parameters:   parsePolicyParameter(policyDef.Parameters),
		IsInitiative: true,
	}
	if policyDef.Description != nil {
		p.Description = *policyDef.Description
	}
	if category, ok := getBuiltinPolicyCategory(policyDef.Metadata); ok {
		p.Category = category
	}
	return &p, nil
}

func policyDefToPolicy(policyDef policy.Definition) (*Policy, error) {
	staticEffect := getBuiltinPolicyStaticEffect(policyDef.PolicyRule)
	if staticEffect == nil {
		return nil, fmt.Errorf("the effect is not found within the policy definition: '%s'", *policyDef.DisplayName)
	}

	p := Policy{
		DisplayName: *policyDef.DisplayName,
		Effect:      *staticEffect,
		ResourceID:  *policyDef.ID,
		Parameters:  parsePolicyParameter(policyDef.Parameters),
	}
	if policyDef.Description != nil {
		p.Description = *policyDef.Description
	}
	if category, ok := getBuiltinPolicyCategory(policyDef.Metadata); ok {
		p.Category = category
	}
	return &p, nil
}

func getBuiltinPolicyCategory(metadata interface{}) (string, bool) {
	m, ok := metadata.(map[string]interface{})
	if !ok {
		return "", false
	}
	category, ok := m["category"]
	if !ok {
		return "", false
	}
	categoryStr, ok := category.(string)
	if !ok {
		return "", false
	}
	return categoryStr, true
}

func getBuiltinPolicyStaticEffect(policyRule interface{}) *string {
	ifAndThen, ok := policyRule.(map[string]interface{})
	if !ok {
		return nil
	}
	then, ok := ifAndThen["then"]
	if !ok {
		return nil
	}
	thenMap := then.(map[string]interface{})
	effect, ok := thenMap["effect"]
	if !ok {
		return nil
	}
	effectStr := effect.(string)
	return &effectStr
}
