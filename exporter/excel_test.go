package exporter

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestReadPoliciesFromExcel(t *testing.T) {
	if SkipTest() {
		return
	}
	base, err := getDir()
	assert.NoError(t, err)
	policies, err := ReadPoliciesFromExcel(filepath.Join(base, ""), "", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, policies)
	_ = PrettyPrint(policies)
}

func TestReadPolicySetParametersFromExcel(t *testing.T) {
	if SkipTest() {
		return
	}
	base, err := getDir()
	assert.NoError(t, err)
	params, err := ReadPolicySetParametersFromExcel(filepath.Join(base, "Azure Cloud Foundation - Baseline Policies.xlsx"), "")
	assert.NoError(t, err)
	assert.NotEmpty(t, params)
	_ = PrettyPrint(params)
}

func getDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Home", "Temp", "azure"), nil
}
