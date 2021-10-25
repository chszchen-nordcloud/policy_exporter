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

	// CellValueNotApplied is a magic value used for values of management group columns. Note that this value
	// is used for the entire cell instead of values for separate parameters. It means the policy is not used
	// for that management group.
	CellValueNotApplied = "n/a"

	// CellValueEnabled is a magic value used to indicate that a policy will be deployed for a management group.
	// Policy parameters will fallback to their default values if they have one. If there are parameters without
	// default values, then the value should be provided explicitly instead.
	CellValueEnabled = "YES"
)

// Reader definitions for obsolete files are used for old baseline file which is exported from Google Doc.
// The obsolete file format is supposed to be static as the file is no longer in active maintenance.
// The other readers are used to read the newly created intermediate excel file which is used to collect data provided
// manually.
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

	// Didn't bother to replace magic numbers as the file to parse is obsolete.
	obsoleteBuiltinPoliciesSheetReader = policySheetReader{
		SheetName: SheetNameBuiltinPolicies,
		RowToPolicy: func(values namedCells) (*Policy, error) {
			row := values.values
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

	// Didn't bother to replace magic numbers as the file to parse is obsolete.
	obsoleteASCPolicyParametersSheetReader = policyParameterSheetReader{
		SheetName: SheetNameAscParameters,
		RowToPolicyParameter: func(values namedCells) (*PolicyParameter, error) {
			row := values.values
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

func ReadPolicyDefinitionFromObsoleteExcel(sourceFilePath string, managementGroups []string) (*ExcelPolicyDefinition, error) {
	result, err := readPolicyDefinitionFromExcel(sourceFilePath, managementGroups, ExcelPolicyDefinitionReader{
		BuiltInPoliciesReader:     &obsoleteBuiltinPoliciesSheetReader,
		ASCPolicyParametersReader: &obsoleteASCPolicyParametersSheetReader,
	})
	if err != nil {
		return nil, err
	}
	parameters := make([]UniqueResource, len(result.ASCPolicySetParameters))
	for i := range result.ASCPolicySetParameters {
		parameters[i] = &result.ASCPolicySetParameters[i]
	}
	uniqueResources := Unique(parameters)
	uniqueParameters := make([]PolicyParameter, len(uniqueResources))
	for i := range uniqueResources {
		uniqueParameters[i] = *(uniqueResources[i].(*PolicyParameter))
	}
	result.ASCPolicySetParameters = uniqueParameters
	return result, nil
}

// ReadPolicyDefinitionFromExcel is mainly used to read data that has to be provided manually. Because there is no need
// to parse the information that is automatically generated. Currently, management group columns and justification,
// cost impact are read from the file.
func ReadPolicyDefinitionFromExcel(sourceFilePath string, managementGroups []string) (*ExcelPolicyDefinition, error) {
	return readPolicyDefinitionFromExcel(sourceFilePath, managementGroups, ExcelPolicyDefinitionReader{
		BuiltInPoliciesReader:     &builtinPoliciesSheetReader,
		CustomPoliciesReader:      &customPoliciesSheetReader,
		ASCPolicyParametersReader: &ascPolicyParametersSheetReader,
	})
}

func readPolicyDefinitionFromExcel(
	sourceFilePath string, managementGroups []string, excelReader ExcelPolicyDefinitionReader,
) (*ExcelPolicyDefinition, error) {
	f, err := excelize.OpenFile(sourceFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file due to: %w", err)
	}

	var result ExcelPolicyDefinition

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

type ExcelPolicyDefinition struct {
	BuiltInPolicies        []Policy
	CustomPolicies         []Policy
	ASCPolicySetParameters []PolicyParameter
}

type policySheetReader struct {
	SheetName       string
	SheetDefinition *sheetDefinition
	RowToPolicy     func(namedCells) (*Policy, error)
}

type policyParameterSheetReader struct {
	SheetName            string
	SheetDefinition      *sheetDefinition
	RowToPolicyParameter func(namedCells) (*PolicyParameter, error)
}

func readPoliciesFromSheet(f *excelize.File, managementGroups []string, reader *policySheetReader) ([]Policy, error) {
	rows, err := reader.readRows(f, managementGroups)
	if err != nil {
		return nil, err
	}

	policies := make([]Policy, 0, len(rows))
	for _, row := range rows {
		policy, err := reader.RowToPolicy(row)
		if err != nil {
			return nil, err
		}
		policies = append(policies, *policy)
	}
	return policies, nil
}

func readPolicyParametersFromSheet(f *excelize.File, managementGroups []string, reader *policyParameterSheetReader) ([]PolicyParameter, error) {
	rows, err := reader.readRows(f, managementGroups)
	if err != nil {
		return nil, err
	}

	parameters := make([]PolicyParameter, 0, len(rows))
	for _, row := range rows {
		parameter, err := reader.RowToPolicyParameter(row)
		if err != nil {
			return nil, err
		}
		parameters = append(parameters, *parameter)
	}
	return parameters, nil
}

func rowToPolicyParameter(values namedCells) (*PolicyParameter, error) {
	typesStr, err := values.MustGet(ColumnParameterTypes)
	if err != nil {
		return nil, err
	}
	parameterType := parseSingleParameterRawValue(typesStr)

	rootManagementGroup := values.GetRootManagementGroup()
	managementGroupToValueStr := values.GetDynamicColumnValues()
	managementGroupToValue := make(map[string]interface{})
	for mgmtGroupName, valueStr := range managementGroupToValueStr {
		valueStr := strings.TrimSpace(valueStr)
		if valueStr == "" || valueStr == CellValueNotApplied {
			continue
		}
		if strings.EqualFold(valueStr, CellValueEnabled) {
			defaultStr, err := values.MustGet(ColumnDefaultValues)
			if err != nil {
				return nil, err
			}
			defaultValue := parseSingleParameterRawValue(defaultStr)
			managementGroupToValue[mgmtGroupName] = defaultValue
		} else {
			converted, err := parseSingleParameterValue(parameterType, valueStr)
			if err != nil {
				return nil, err
			}
			managementGroupToValue[mgmtGroupName] = converted
		}
	}
	if v, ok := managementGroupToValue[rootManagementGroup]; ok {
		managementGroupToValue = make(map[string]interface{})
		managementGroupToValue[rootManagementGroup] = v
	}

	internalName, err := values.MustGet(ColumnReferenceID)
	if err != nil {
		return nil, err
	}

	justification, _ := values.Get(ColumnJustification)
	costImpact, _ := values.Get(ColumnCostImpact)

	parameter := PolicyParameter{
		InternalName:     internalName,
		Justification:    justification,
		CostImpact:       costImpact,
		ManagementGroups: managementGroupToValue,
	}
	return &parameter, nil
}

func rowToPolicy(values namedCells) (*Policy, error) {
	displayName, err := values.MustGet(ColumnDisplayName)
	if err != nil {
		return nil, err
	}

	category, _ := values.Get(ColumnCategory)

	typesStr, ok := values.Get(ColumnParameterTypes)
	if !ok {
		fmt.Printf("policy '%s' has no parameters, skip parsing parameter values\n", displayName)
		return &Policy{
			DisplayName: displayName,
			Category:    category,
		}, nil
	}
	parameterTypes := parsePolicyParameterRawValues(typesStr)

	toAttachment := func(v string) (*Attachment, error) {
		valueStr := strings.TrimSpace(v)
		if valueStr == "" || valueStr == CellValueNotApplied {
			return nil, nil
		}
		var attachment Attachment
		if !strings.EqualFold(valueStr, CellValueEnabled) {
			converted, err := parsePolicyParameterValues(parameterTypes, valueStr)
			if err != nil {
				return nil, err
			}
			attachment = Attachment{
				Parameters: converted,
			}
			if effect, ok := converted["effect"]; ok {
				attachment.Effect = effect.(string)
			}
		}
		attachment.Enabled = true
		attachment.Location = "variables('managedIdentityLocation')]"
		return &attachment, nil
	}

	attachments := make(map[string]Attachment)
	rootManagementGroup := values.GetRootManagementGroup()
	managementGroupToValueStr := values.GetDynamicColumnValues()
	for mgmtGroupName, valueStr := range managementGroupToValueStr {
		attachment, err := toAttachment(valueStr)
		if err != nil {
			return nil, err
		}
		if attachment != nil {
			attachments[mgmtGroupName] = *attachment
		}
	}
	if v, ok := attachments[rootManagementGroup]; ok {
		attachments = make(map[string]Attachment)
		attachments[rootManagementGroup] = v
	}

	justification, _ := values.Get(ColumnJustification)
	costImpact, _ := values.Get(ColumnCostImpact)

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

// all possible types are listed according to [this](https://docs.microsoft.com/en-us/azure/governance/policy/concepts/definition-structure#parameters)
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

// if we know there can only be one value.
func parseSingleParameterRawValue(s string) string {
	for _, v := range parsePolicyParameterRawValues(s) {
		return v
	}
	return ""
}

// turn cell value into key value pairs
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

func (r *policyParameterSheetReader) readRows(f *excelize.File, dynamicColumns []string) ([]namedCells, error) {
	return readRows(f, r.SheetName, r.SheetDefinition, dynamicColumns)
}

func (r *policySheetReader) readRows(f *excelize.File, dynamicColumns []string) ([]namedCells, error) {
	return readRows(f, r.SheetName, r.SheetDefinition, dynamicColumns)
}

func readRows(f *excelize.File, sheetName string, definition *sheetDefinition, dynamicColumns []string) ([]namedCells, error) {
	var result []namedCells
	if definition == nil {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			return nil, err
		}
		result = make([]namedCells, 0, len(rows)-1)
		for _, row := range rows[1:] {
			result = append(result, namedCells{values: row})
		}
	} else {
		sheet, err := newExcelSheet(f, *definition, dynamicColumns)
		if err != nil {
			return nil, err
		}
		result, err = sheet.GetRows()
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

type namedCells struct {
	columns *columns
	values  []string
}

// GetDynamicColumnValues returns a column name to value mapping in which only dynamic columns are included.
// 'dynamic columns' are columns that may vary based on user inputs and cannot be hardcoded.
func (c *namedCells) GetDynamicColumnValues() map[string]string {
	result := make(map[string]string)
	for idx := c.columns.dynamicColumnStart; idx < c.columns.dynamicColumnEnd; idx++ {
		key := c.columns.headers[idx]
		result[key] = c.values[idx]
	}
	return result
}

func (c *namedCells) GetRootManagementGroup() string {
	return c.columns.headers[c.columns.dynamicColumnStart]
}

// Get returns a flag indicating whether the value is provided for the given column.
func (c *namedCells) Get(column string) (string, bool) {
	idx, ok := c.columns.indexes[column]
	if !ok {
		return "", false
	}
	if len(c.values) < idx+1 {
		return "", false
	}
	return c.values[idx], true
}

// MustGet returns error if value is not provided for the given column.
func (c *namedCells) MustGet(column string) (string, error) {
	v, ok := c.Get(column)
	if !ok {
		return "", fmt.Errorf("value for column '%s' does not exist in row %v", column, c.values)
	}
	return v, nil
}

func (c *namedCells) ID() interface{} {
	internalName, ok := c.Get(ColumnReferenceID)
	if ok {
		return internalName
	}
	displayName, ok := c.Get(ColumnDisplayName)
	if ok {
		return displayName
	}
	return nil
}
