package exporter

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestReadFromYAML(t *testing.T) {
	if SkipTest() {
		return
	}

	base := TestResourceDir()
	result, err := ReadPolicyDefinitionFromYAML(filepath.Join(base, "example_policies.yaml"))
	assert.NoError(t, err)

	assert.NotEmpty(t, result.BuiltInPolicies)

	policy := result.BuiltInPolicies[0]
	assert.NotEqual(t, "", policy.DisplayName)
	assert.NotEqual(t, "", policy.Description)
	assert.NotEqual(t, "", policy.ResourceID)

	_ = PrettyPrint(result)
}
