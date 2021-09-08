package exporter

import (
	"errors"
	"fmt"
	"os"
)

type Config struct {
	// Required, will read from environment variable if not set
	SubscriptionID string

	ASCPolicySetName string

	ManagementGroupName string
	ExcelFilePath       string
	YAMLFilePath        string

	// Required, management groups that appear as columns in exported files
	ManagementGroups []string

	// Default to current dir, directory for exported files
	TargetDir string

	// Required, needed to resolve the category for policies
	LocalLandingZoneRepoDir string
}

func (c *Config) Validate() error {
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
