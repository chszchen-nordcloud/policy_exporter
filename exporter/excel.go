package exporter

import (
	"github.com/360EntSecGroup-Skylar/excelize"
)

const (
	SHEET_NAME_BUILTIN_POLICIES = "Built-in policies"
	SHEET_NAME_CUSTOM_POLICIES  = "Nordcloud custom policies"
	SHEET_NAME_ASC_PARAMETERS   = "Security policies"
)

func ReadPoliciesFromExcel(filepath string, builtInPolicySheetName string, customPolicySheetName string) ([]Policy, error) {
	f, err := excelize.OpenFile(filepath)
	if err != nil {
		return nil, err
	}

	if builtInPolicySheetName == "" {
		builtInPolicySheetName = SHEET_NAME_BUILTIN_POLICIES
	}
	if customPolicySheetName == "" {
		customPolicySheetName = SHEET_NAME_CUSTOM_POLICIES
	}

	rows := f.GetRows(builtInPolicySheetName)
	policies := parseBuiltInPolicies(rows)

	rows = f.GetRows(customPolicySheetName)
	policies = append(policies, parseCustomPolicies(rows)...)

	return policies, nil
}

func parseCustomPolicies(rows [][]string) []Policy {
	result := make([]Policy, 0, len(rows))
	for _, row := range rows[1:] {
		result = append(result, Policy{
			DisplayName: row[0],
			Description: row[6],
		})
	}
	return result
}

func parseBuiltInPolicies(rows [][]string) []Policy {
	result := make([]Policy, 0, len(rows))
	for _, row := range rows[1:] {
		result = append(result, Policy{
			Justification: row[7],
		})
	}
	return result
}

func ReadPolicySetParametersFromExcel(filepath string, ascPolicySheetName string) ([]PolicyParameter, error) {
	f, err := excelize.OpenFile(filepath)
	if err != nil {
		return nil, err
	}

	if ascPolicySheetName == "" {
		ascPolicySheetName = SHEET_NAME_ASC_PARAMETERS
	}

	rows := f.GetRows(ascPolicySheetName)
	params := parsePolicyParameters(rows)
	return params, nil
}

func parsePolicyParameters(rows [][]string) []PolicyParameter {
	result := make([]PolicyParameter, 0, len(rows))
	for _, row := range rows[1:] {
		result = append(result, PolicyParameter{
			Justification: row[7],
			CostImpact:    row[8],
		})
	}
	return result
}
