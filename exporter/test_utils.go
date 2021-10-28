package exporter

import (
	"encoding/json"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func SkipTest() bool {
	v := os.Getenv("TEST")
	return strings.ToLower(v) != "true"
}

func PrettyPrint(v interface{}) error {
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return err
	}
	color.Green("%s\n", string(b))
	return nil
}

func TestResourceDir() string {
	return "../test_resources"
}

func getConfigForTest(t *testing.T) *Config {
	resourceDir := "test_resources"
	configFilePath := filepath.Join(resourceDir, "example_config.yaml")
	config, err := buildConfig(&configFilePath, []string{resourceDir})
	assert.NoError(t, err)

	err = config.Validate()
	assert.NoError(t, err)

	return config
}
