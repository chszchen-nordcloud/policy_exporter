package exporter

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"os"
	"path/filepath"
)

const ExportedPolicyJsonParameterFileName = "governance-policy-parameters.json"

func ExportPoliciesAsJSON(policies []Policy, targetDir string) error {
	categoryByName := make(map[string]*Category)
	for _, policy := range policies {
		if len(policy.ManagementGroups) == 0 {
			continue
		}
		categoryName := policy.Category
		if categoryName == "" {
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

	var m JSONObject
	policyParameterFile := filepath.Join(targetDir, ExportedPolicyJsonParameterFileName)
	if _, err := os.Stat(policyParameterFile); err == nil {
		tmpl, err := NewJSONObjectFromFile(policyParameterFile)
		if err != nil {
			return err
		}
		tmpl["category"] = categories
		m = tmpl
	} else {
		m = map[string]interface{}{"category": categories}
	}

	b, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		return err
	}

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

func ExportPolicySetParametersAsJSON(parameters []PolicyParameter, subscriptions []string, targetDir string) error {
	for _, subscription := range subscriptions {
		err := doExportPolicySetParametersAsJSON(parameters, subscription, getTargetPolicyParameterJSONFile(targetDir, subscription))
		if err != nil {
			return err
		}
	}
	return nil
}

func doExportPolicySetParametersAsJSON(parameters []PolicyParameter, managementGroup string, targetFile string) error {
	parametersToExport := make(map[string]PolicyParameterValue)
	for _, param := range parameters {
		if len(param.ManagementGroups) == 0 {
			continue
		}
		v, ok := param.ManagementGroups[managementGroup]
		if ok {
			parametersToExport[param.InternalName] = PolicyParameterValue{
				Value: v,
			}
		}
	}

	var m JSONObject
	if _, err := os.Stat(targetFile); err == nil {
		tmpl, err := NewJSONObjectFromFile(targetFile)
		if err != nil {
			return err
		}
		tmpl["parameters"] = parametersToExport
		m = tmpl
	} else {
		m = map[string]interface{}{"parameters": parametersToExport}
	}

	b, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		return err
	}

	color.Green("Write to '%s'", targetFile)
	return os.WriteFile(targetFile, b, 0600)
}

func getTargetPolicyParameterJSONFile(targetDir string, subscription string) string {
	return filepath.Join(targetDir, fmt.Sprintf("%s.json", subscription))
}
