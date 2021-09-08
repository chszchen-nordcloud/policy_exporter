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
	_ = PrettyPrint(result)
}
