package exporter

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func ExportPoliciesAsJSON(policies []Policy, landingZoneRepoPath string, targetDir string) error {
	governancePoliciesDir := filepath.Join(landingZoneRepoPath, "Governance", "Policies")
	templateDir := filepath.Join(governancePoliciesDir, "Templates")
	categoryByFileName, err := getCategoryMappings(templateDir, "")
	if err != nil {
		return err
	}
	color.Green("Built category mappings from directory '%s'", templateDir)

	categoryByName := make(map[string]*Category)
	for _, policy := range policies {
		categoryName, ok := categoryByFileName[policy.DisplayName]
		if !ok {
			categoryName = "Unknown"
		}
		category, ok := categoryByName[categoryName]
		if !ok {
			category = &Category{
				Name: categoryName,
			}
			categoryByName[categoryName] = category
		}
		category.Policies = append(category.Policies, policy)
	}

	categories := make([]Category, 0, len(categoryByName))
	for _, category := range categoryByName {
		categories = append(categories, *category)
	}
	PrintCategorySummary(categoryByName)

	m := map[string]interface{}{"category": categories}
	b, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		return err
	}

	policyParameterFile := filepath.Join(targetDir, "governance-policy-parameters.json")
	color.Green("Write to '%s'", policyParameterFile)
	return os.WriteFile(policyParameterFile, b, 0600)
}

func PrintCategorySummary(categoryByName map[string]*Category) {
	c := color.New(color.FgGreen)
	c.Print("Policies found")
	for _, category := range categoryByName {
		c.Printf(" [%s: %d]", category.Name, len(category.Policies))
	}
	c.Printf("\n")
}

func getCategoryMappings(templateDir string, category string) (map[string]string, error) {
	files, err := ioutil.ReadDir(templateDir)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, f := range files {
		switch {
		case f.IsDir():
			if category == "" {
				category = f.Name()
			}
			innerMappings, err := getCategoryMappings(filepath.Join(templateDir, f.Name()), category)
			if err != nil {
				return nil, err
			}
			for k, v := range innerMappings {
				result[k] = v
			}
		case strings.HasSuffix(f.Name(), "."):
			fmt.Printf("Ignore file %s/%s", templateDir, f.Name())
		default:
			result[f.Name()] = category
		}
	}
	return result, nil
}

func ExportPolicySetParametersAsJSON(parameters []PolicyParameter, managementGroup string, targetDir string) error {
	parametersToExport := make(map[string]PolicyParameterValue)
	for _, param := range parameters {
		effect, ok := param.ManagementGroups[managementGroup]
		if !ok {
			effect = "Disabled"
		}
		parametersToExport[param.InternalName] = PolicyParameterValue{
			Value: effect,
		}
	}

	m := map[string]interface{}{"parameters": parametersToExport}
	b, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		return err
	}

	parameterFile := filepath.Join(targetDir, fmt.Sprintf("ASC_policy_%s.json", managementGroup))
	color.Green("Write to '%s'", parameterFile)
	return os.WriteFile(parameterFile, b, 0600)
}
