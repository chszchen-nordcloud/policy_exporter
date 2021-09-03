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

func (az *AzureAPI) GetPolicySetParameters(ctx context.Context, policySetName string) (PolicySetParameters, error) {
	policySet, err := az.policySetAPI.GetBuiltIn(ctx, policySetName)
	if err != nil {
		return nil, err
	}
	var result PolicySetParameters
	for internalName, paramDef := range policySet.Parameters {
		if v, ok := paramDef.DefaultValue.(string); ok {
			result[internalName] = PolicySetParameterValue{ Value: v}
		} else {
			return nil, UnexpectedValueError(internalName, paramDef.DefaultValue)
		}
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
		result = append(result, Policy {
			DisplayName: *policyDef.DisplayName,
			ResourceID: *policyDef.ID,
		})
	}
	return result, nil
}