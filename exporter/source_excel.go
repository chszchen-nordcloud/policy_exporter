package exporter

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cast"
	"github.com/xuri/excelize/v2"
	"strings"
)

const (
	SheetNameBuiltinPolicies = "Built-in policies"
	SheetNameCustomPolicies  = "Nordcloud custom policies"
	SheetNameAscParameters   = "Security policies"

	CellValueNotApplied = "n/a"
)

var (
	builtinPoliciesSheetReader = policySheetReader{
		SheetName:       SheetNameBuiltinPolicies,
		SheetDefinition: &SheetBuiltInPolicy,
		RowToPolicy:     rowToPolicy,
	}
	customPoliciesSheetReader = policySheetReader{
		SheetName:       SheetNameCustomPolicies,
		SheetDefinition: &SheetCustomPolicy,
		RowToPolicy:     rowToPolicy,
	}
	ascPolicyParametersSheetReader = policyParameterSheetReader{
		SheetName:            SheetNameAscParameters,
		SheetDefinition:      &SheetASCParameters,
		RowToPolicyParameter: rowToPolicyParameter,
	}
	obsoleteBuiltinPoliciesSheetReader = policySheetReader{
		SheetName: SheetNameBuiltinPolicies,
		RowToPolicy: func(row []string, _ *columns) (*Policy, error) {
			var justification string
			if len(row) >= 9 {
				justification = row[8]
			}
			policy := Policy{
				DisplayName:   row[1],
				Category:      row[7],
				Justification: justification,
			}
			return &policy, nil
		},
	}
	obsoleteASCPolicyParametersSheetReader = policyParameterSheetReader{
		SheetName: SheetNameAscParameters,
		RowToPolicyParameter: func(row []string, _ *columns) (*PolicyParameter, error) {
			var justification, costImpact string
			if len(row) >= 8 {
				justification = row[7]
			}
			if len(row) >= 9 {
				costImpact = row[8]
			}
			return &PolicyParameter{
				InternalName:  row[0],
				Justification: justification,
				CostImpact:    costImpact,
			}, nil
		},
	}
)

func ReadPolicyDefinitionFromObsoleteExcel(sourceFilePath string, managementGroups []string) (*excelPolicyDefinition, error) {
	return readPolicyDefinitionFromExcel(sourceFilePath, managementGroups, ExcelPolicyDefinitionReader{
		BuiltInPoliciesReader:     &obsoleteBuiltinPoliciesSheetReader,
		ASCPolicyParametersReader: &obsoleteASCPolicyParametersSheetReader,
	})
}

func ReadPolicyDefinitionFromExcel(sourceFilePath string, managementGroups []string) (*excelPolicyDefinition, error) {
	return readPolicyDefinitionFromExcel(sourceFilePath, managementGroups, ExcelPolicyDefinitionReader{
		BuiltInPoliciesReader:     &builtinPoliciesSheetReader,
		CustomPoliciesReader:      &customPoliciesSheetReader,
		ASCPolicyParametersReader: &ascPolicyParametersSheetReader,
	})
}

// ReadPolicyDefinitionFromExcel reads the following from excel file,
// * 'Justification' of builtin policies.
// * Custom policies.
// * 'Justification' and 'CostImpact' of ASC policy parameters.
func readPolicyDefinitionFromExcel(
	sourceFilePath string, managementGroups []string, excelReader ExcelPolicyDefinitionReader,
) (*excelPolicyDefinition, error) {
	f, err := excelize.OpenFile(sourceFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file due to: %w", err)
	}

	var result excelPolicyDefinition

	if excelReader.BuiltInPoliciesReader != nil {
		policies, err := readPoliciesFromSheet(f, managementGroups, excelReader.BuiltInPoliciesReader)
		if err != nil {
			return nil, err
		}
		result.BuiltInPolicies = policies
	}

	if excelReader.CustomPoliciesReader != nil {
		policies, err := readPoliciesFromSheet(f, managementGroups, excelReader.CustomPoliciesReader)
		if err != nil {
			return nil, err
		}
		result.CustomPolicies = policies
	}

	if excelReader.ASCPolicyParametersReader != nil {
		parameters, err := readPolicyParametersFromSheet(f, managementGroups, excelReader.ASCPolicyParametersReader)
		if err != nil {
			return nil, err
		}
		result.ASCPolicySetParameters = parameters
	}

	return &result, nil
}

type ExcelPolicyDefinitionReader struct {
	BuiltInPoliciesReader     *policySheetReader
	CustomPoliciesReader      *policySheetReader
	ASCPolicyParametersReader *policyParameterSheetReader
}

type excelPolicyDefinition struct {
	BuiltInPolicies        []Policy
	CustomPolicies         []Policy
	ASCPolicySetParameters []PolicyParameter
}

type policySheetReader struct {
	SheetName       string
	SheetDefinition *sheetDefinition
	RowToPolicy     func(values []string, columns *columns) (*Policy, error)
}

type policyParameterSheetReader struct {
	SheetName            string
	SheetDefinition      *sheetDefinition
	RowToPolicyParameter func(values []string, columns *columns) (*PolicyParameter, error)
}

func readPoliciesFromSheet(f *excelize.File, managementGroups []string, reader *policySheetReader) ([]Policy, error) {
	rows, err := f.GetRows(reader.SheetName)
	if err != nil {
		return nil, err
	}

	var columns *columns
	if reader.SheetDefinition != nil {
		_, cols, err := reader.SheetDefinition.GetHeaders(managementGroups)
		if err != nil {
			return nil, err
		}
		columns = cols
	}

	policies := make([]Policy, 0, len(rows))
	for _, row := range rows[1:] {
		policy, err := reader.RowToPolicy(row, columns)
		if err != nil {
			return nil, err
		}
		policies = append(policies, *policy)
	}
	return policies, nil
}

func readPolicyParametersFromSheet(f *excelize.File, managementGroups []string, reader *policyParameterSheetReader) ([]PolicyParameter, error) {
	rows, err := f.GetRows(reader.SheetName)
	if err != nil {
		return nil, err
	}

	var columns *columns
	if reader.SheetDefinition != nil {
		_, cols, err := reader.SheetDefinition.GetHeaders(managementGroups)
		if err != nil {
			return nil, err
		}
		columns = cols
	}

	parameters := make([]PolicyParameter, 0, len(rows))
	for _, row := range rows[1:] {
		parameter, err := reader.RowToPolicyParameter(row, columns)
		if err != nil {
			return nil, err
		}
		parameters = append(parameters, *parameter)
	}
	return parameters, nil
}

func rowToPolicyParameter(values []string, columns *columns) (*PolicyParameter, error) {
	typesStr, err := columns.MustGetValue(values, ColumnParameterTypes)
	if err != nil {
		return nil, fmt.Errorf("row values does not contain parameter types: %v", values)
	}
	parameterType := parseSingleParameterRawValue(typesStr)

	managementGroupToValueStr := columns.GetDynamicColumnValues(values)
	managementGroupToValue := make(map[string]interface{}, len(managementGroupToValueStr))
	for mgmtGroupName, valueStr := range managementGroupToValueStr {
		valueStr := strings.TrimSpace(valueStr)
		if valueStr == "" || valueStr == CellValueNotApplied {
			continue
		}
		converted, err := parseSingleParameterValue(parameterType, valueStr)
		if err != nil {
			return nil, err
		}
		managementGroupToValue[mgmtGroupName] = converted
	}

	internalName, err := columns.MustGetValue(values, ColumnReferenceID)
	if err != nil {
		return nil, err
	}

	justification, _ := columns.GetValue(values, ColumnJustification)
	costImpact, _ := columns.GetValue(values, ColumnCostImpact)

	parameter := PolicyParameter{
		InternalName:     internalName,
		Justification:    justification,
		CostImpact:       costImpact,
		ManagementGroups: managementGroupToValue,
	}
	return &parameter, nil
}

func rowToPolicy(values []string, columns *columns) (*Policy, error) {
	displayName, err := columns.MustGetValue(values, ColumnDisplayName)
	if err != nil {
		return nil, err
	}

	category, _ := columns.GetValue(values, ColumnCategory)

	typesStr, ok := columns.GetValue(values, ColumnParameterTypes)
	if !ok {
		fmt.Printf("policy '%s' has no parameters, skip parsing parameter values\n", displayName)
		return &Policy{
			DisplayName: displayName,
			Category:    category,
		}, nil
	}
	parameterTypes := parsePolicyParameterRawValues(typesStr)

	attachments := make(map[string]Attachment)
	managementGroupToValueStr := columns.GetDynamicColumnValues(values)
	for mgmtGroupName, valueStr := range managementGroupToValueStr {
		valueStr := strings.TrimSpace(valueStr)
		if valueStr == "" || valueStr == CellValueNotApplied {
			continue
		}
		converted, err := parsePolicyParameterValues(parameterTypes, valueStr)
		if err != nil {
			return nil, err
		}
		attachment := Attachment{
			Parameters: converted,
		}
		if effect, ok := converted["effect"]; ok {
			attachment.Effect = effect.(string)
		}
		attachments[mgmtGroupName] = attachment
	}

	justification, _ := columns.GetValue(values, ColumnJustification)
	costImpact, _ := columns.GetValue(values, ColumnCostImpact)

	policy := Policy{
		Category:         category,
		DisplayName:      displayName,
		Justification:    justification,
		CostImpact:       costImpact,
		ManagementGroups: attachments,
	}
	return &policy, nil
}

func parsePolicyParameterValues(parameterTypes map[string]string, value string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	rawValues := parsePolicyParameterRawValues(value)
	for name, rawValue := range rawValues {
		parameterType, ok := parameterTypes[name]
		if !ok {
			return nil, fmt.Errorf("the parameter type of '%s' is not provided", name)
		}
		converted, err := parseSingleParameterValue(parameterType, rawValue)
		if err != nil {
			return nil, err
		}
		result[name] = converted
	}
	return result, nil
}

func parseSingleParameterValue(parameterType string, s string) (interface{}, error) {
	value := parseSingleParameterRawValue(s)
	var converted interface{}
	switch strings.ToLower(parameterType) {
	case "integer":
		converted = cast.ToInt(value)
	case "float":
		converted = cast.ToFloat64(value)
	case "boolean":
		converted = cast.ToBool(value)
	case "array":
		arr := make([]interface{}, 0, 4)
		err := json.Unmarshal([]byte(value), &arr)
		if err != nil {
			return nil, err
		}
		converted = arr
	case "object":
		m := make(map[string]interface{})
		err := json.Unmarshal([]byte(value), &m)
		if err != nil {
			return nil, err
		}
		converted = m
	case "string":
		converted = value
	default:
		return nil, fmt.Errorf("unsupported parameter type '%s'", parameterType)
	}

	return converted, nil
}

func parseSingleParameterRawValue(s string) string {
	for _, v := range parsePolicyParameterRawValues(s) {
		return v
	}
	return ""
}

func parsePolicyParameterRawValues(s string) map[string]string {
	lines := strings.Split(s, "\n")
	m := make(map[string]string, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.LastIndex(line, ":")
		k, v := "", line
		if idx != -1 {
			k = strings.TrimSpace(line[0:idx])
			v = strings.TrimSpace(line[idx+1:])
		}
		m[k] = v
	}
	return m
}
