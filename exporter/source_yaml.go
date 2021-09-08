package exporter

import (
	"gopkg.in/yaml.v3"
	"os"
)

type YAMLPolicyDefinition struct {
	BuiltInPolicies        []Policy          `yaml:"BuiltInPolicies"`
	CustomPolicies         []Policy          `yaml:"CustomPolicies"`
	ASCPolicySetParameters []PolicyParameter `yaml:"ASCPolicySetParameters"`
	ManagementGroups       []string          `yaml:"ManagementGroups"`
}

func ReadPolicyDefinitionFromYAML(sourcePath string) (*YAMLPolicyDefinition, error) {
	b, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, err
	}
	var result YAMLPolicyDefinition
	err = yaml.Unmarshal(b, &result)
	for i := range result.BuiltInPolicies {
		result.BuiltInPolicies[i].Optional = false
	}
	return &result, err
}
