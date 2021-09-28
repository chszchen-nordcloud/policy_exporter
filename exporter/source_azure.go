package exporter

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/policy"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
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
			Optional:     false,
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
	page, err := az.policyAPI.ListByManagementGroup(ctx, managementGroupID)
	if err != nil {
		return nil, err
	}

	var result []Policy
	for list := policy.NewDefinitionListResultIterator(page); list.NotDone(); err = list.NextWithContext(ctx) {
		if err != nil {
			return nil, err
		}
		policyDef := list.Value()
		p := Policy{
			DisplayName: *policyDef.DisplayName,
			ResourceID:  *policyDef.ID,
			Parameters:  parsePolicyParameter(policyDef.Parameters),
			Optional:    false,
		}
		if policyDef.Description != nil {
			p.Description = *policyDef.Description
		}
		result = append(result, p)
	}
	return result, nil
}
