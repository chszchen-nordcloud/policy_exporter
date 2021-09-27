package exporter

import (
	"fmt"
	"github.com/xuri/excelize/v2"
)

const (
	SHEET_NAME_BUILTIN_POLICIES = "Built-in policies"
	SHEET_NAME_CUSTOM_POLICIES  = "Nordcloud custom policies"
	SHEET_NAME_ASC_PARAMETERS   = "Security policies"
)

type ExcelPolicyDefinition struct {
	BuiltInPolicies        []Policy
	CustomPolicies         []Policy
	ASCPolicySetParameters []PolicyParameter
}

// ReadPolicyDefinitionFromExcel reads the following from excel file,
// * 'Justification' of builtin policies.
// * Custom policies.
// * 'Justification' and 'CostImpact' of ASC policy parameters.
func ReadPolicyDefinitionFromExcel(
	sourceFilePath string, builtInPolicySheetName string, customPolicySheetName string, ascPolicySheetName string,
) (*ExcelPolicyDefinition, error) {
	f, err := excelize.OpenFile(sourceFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file due to: %w", err)
	}

	if builtInPolicySheetName == "" {
		builtInPolicySheetName = SHEET_NAME_BUILTIN_POLICIES
	}
	if customPolicySheetName == "" {
		customPolicySheetName = SHEET_NAME_CUSTOM_POLICIES
	}
	if ascPolicySheetName == "" {
		ascPolicySheetName = SHEET_NAME_ASC_PARAMETERS
	}

	var result ExcelPolicyDefinition

	rows, err := f.GetRows(builtInPolicySheetName)
	if err != nil {
		return nil, err
	}
	result.BuiltInPolicies = parseBuiltInPolicies(rows)

	rows, err = f.GetRows(customPolicySheetName)
	if err != nil {
		return nil, err
	}
	result.CustomPolicies = parseCustomPolicies(rows)

	rows, err = f.GetRows(ascPolicySheetName)
	if err != nil {
		return nil, err
	}
	result.ASCPolicySetParameters = parsePolicyParameters(rows)

	return &result, nil
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
