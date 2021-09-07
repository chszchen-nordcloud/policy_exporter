package exporter

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAzureAPI_GetPolicySetParameters(t *testing.T) {
	if SkipTest() {
		return
	}

	az, err := NewAzureAPI("")
	assert.NoError(t, err)

	params, err := az.GetPolicySetParameters(context.Background(), "1f3afdf9-d0c9-4c3d-847f-89da613e70a8")
	assert.NoError(t, err)
	assert.True(t, len(params) > 0)

	param := params[0]
	assert.NotEqual(t, "", param.InternalName)
	assert.NotEqual(t, "", param.DisplayName)
	assert.NotEqual(t, "", param.Description)
	assert.NotEqual(t, "", param.Type)
	assert.NotNil(t, "", param.DefaultValue)

	err = PrettyPrint(params)
	assert.NoError(t, err)
}

func TestAzureAPI_ListBuiltInPolicyByManagementGroup(t *testing.T) {
	if SkipTest() {
		return
	}

	az, err := NewAzureAPI("")
	assert.NoError(t, err)

	policies, err := az.ListBuiltInPolicyByManagementGroup(context.Background(), "Sandbox")
	assert.NoError(t, err)
	assert.True(t, len(policies) > 0)

	var policy Policy
	for _, p := range policies {
		if len(p.Parameters) != 0 {
			policy = p
			break
		}
	}
	assert.NotEqual(t, "", policy.DisplayName)
	assert.NotEqual(t, "", policy.ResourceID)
	assert.NotEqual(t, "", policy.Description)
	assert.NotEmpty(t, policy.Parameters)

	err = PrettyPrint(policies)
	assert.NoError(t, err)
}
