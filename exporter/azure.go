package exporter

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/policy"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"os"
)

const (
	ENV_VAR_SUBSCRIPTION_ID = "AZURE_SUBSCRIPTION_ID"
)

func NewAzureAPIPolicyDefinitionProvider(config Config) (*PolicyDefinitionProvider, error) {
	api, err := NewAzureAPI(config.SubscriptionID)
	if err != nil {
		return nil, err
	}
	return &PolicyDefinitionProvider{
		PolicyReader: func(ctx context.Context) ([]Policy, error) {
			return api.ListBuiltInPolicyByManagementGroup(ctx, config.ManagementGroupName)
		},
		PolicySetParameterReader: func(ctx context.Context) ([]PolicySetParameter, error) {
			return api.GetPolicySetParameters(ctx, config.ASCPolicySetName)
		},
	}, nil
}

type AzureAPI struct {
	authorizer     autorest.Authorizer
	subscriptionID string
	policySetAPI   policy.SetDefinitionsClient
	policyAPI      policy.DefinitionsClient
}

func NewAzureAPI(subscriptionID string) (*AzureAPI, error) {
	var azureAPI AzureAPI
	if subscriptionID == "" {
		if v, ok := os.LookupEnv(ENV_VAR_SUBSCRIPTION_ID); ok {
			subscriptionID = v
		} else {
			return nil, MissingInputError("subscriptionID", "AzureAPI")
		}
	}
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

func (az *AzureAPI) GetPolicySetParameters(ctx context.Context, policySetName string) ([]PolicySetParameter, error) {
	policySet, err := az.policySetAPI.GetBuiltIn(ctx, policySetName)
	if err != nil {
		return nil, err
	}
	result := make([]PolicySetParameter, 0, len(policySet.Parameters))
	for internalName, paramDef := range policySet.Parameters {
		var allowedValues []interface{}
		if paramDef.AllowedValues != nil {
			allowedValues = *paramDef.AllowedValues
		}
		result = append(result, PolicySetParameter{
			InternalName:  internalName,
			DisplayName:   *paramDef.Metadata.DisplayName,
			Description:   *paramDef.Metadata.Description,
			DefaultValue:  paramDef.DefaultValue,
			AllowedValues: allowedValues,
		})
	}
	return result, nil
}

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
		var description string
		if policyDef.Description != nil {
			description = *policyDef.Description
		}
		result = append(result, Policy{
			DisplayName: *policyDef.DisplayName,
			ResourceID:  *policyDef.ID,
			Description: description,
		})
	}
	return result, nil
}
