package exporter

import (
	"errors"
	"fmt"
	"os"
)

const (
	ENV_VAR_SUBSCRIPTION_ID = "AZURE_SUBSCRIPTION_ID"
)

type Config struct {
	// Required, will read from environment variable if not set
	SubscriptionID string `yaml:"SubscriptionID"`

	// Default to '1f3afdf9-d0c9-4c3d-847f-89da613e70a8', used to query ASC policy.
	PolicyQueryASCPolicySetName string `yaml:"PolicyQueryASCPolicySetName"`

	// Default to 'Sandbox', used to query builtin policies.
	PolicyQueryManagementGroupName string `yaml:"PolicyQueryManagementGroupName"`

	// Required for providing justification for policies
	ExcelFilePath string `yaml:"ExcelFilePath"`

	// Required, only builtin policies found in this file will be included in JSON parameter files
	YAMLFilePath string `yaml:"YAMLFilePath"`

	// Required, management groups that appear as columns in exported files
	ManagementGroups []string `yaml:"ManagementGroups"`

	// Default to current dir, directory for exported files
	TargetDir string `yaml:"TargetDir"`

	// Required, needed to resolve the category for policies
	LocalLandingZoneRepoDir string `yaml:"LocalLandingZoneRepoDir"`
}

// Validate validates the config object and provides defaults for fields if necessary.
func (c *Config) Validate() error {
	if _, err := os.Stat(c.ExcelFilePath); err != nil {
		return err
	}
	if _, err := os.Stat(c.YAMLFilePath); err != nil {
		return err
	}
	if c.PolicyQueryManagementGroupName == "" {
		c.PolicyQueryManagementGroupName = "Sandbox"
	}
	if c.PolicyQueryASCPolicySetName == "" {
		c.PolicyQueryASCPolicySetName = "1f3afdf9-d0c9-4c3d-847f-89da613e70a8"
	}
	if c.SubscriptionID == "" {
		if v, ok := os.LookupEnv(ENV_VAR_SUBSCRIPTION_ID); ok {
			c.SubscriptionID = v
		} else {
			return errors.New("subscription is required to retrieve policies from Azure")
		}
	}

	stat, err := os.Stat(c.TargetDir)
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return fmt.Errorf("%s is not a valid directory", c.TargetDir)
	}

	if c.TargetDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		c.TargetDir = wd
	}
	return nil
}
