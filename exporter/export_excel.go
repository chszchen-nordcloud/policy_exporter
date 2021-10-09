package exporter

import (
	"errors"
	"fmt"
	"github.com/spf13/cast"
	"github.com/xuri/excelize/v2"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Prefer string index over hardcoded magic integers.
const (
	ColumnDisplayName    = "DisplayName"
	ColumnPossibleValues = "Parameters: Possible values"
	ColumnDefaultValues  = "Default values"
	ColumnDescription    = "Description"
	ColumnCategory       = "Category"
	ColumnPolicyType     = "Policy Type"
	ColumnResourceID     = "ResourceID"
	ColumnJustification  = "Justification"
	ColumnCostImpact     = "Cost Impact"
	ColumnReferenceID    = "Reference ID"
	ColumnParameterTypes = "Parameter Types"
)

var (
	SheetBuiltInPolicy = sheetDefinition{
		Name:  SheetNameBuiltinPolicies,
		Order: 0,
		Columns: []string{
			ColumnDisplayName, ColumnPossibleValues, ColumnDefaultValues, ColumnDescription, ColumnCategory,
			ColumnPolicyType, ColumnResourceID, ColumnJustification, ColumnCostImpact,
			ColumnParameterTypes,
		},
		DynamicColumnIndex: ColumnDefaultValues,
		GetRows: func(obj interface{}) []partialRow {
			policies := obj.([]Policy)

			rows := make([]partialRow, 0, len(policies))
			for _, policy := range policies {
				row := builtInPolicyToRowValues(policy)
				rows = append(rows, row)
			}
			return rows
		},
	}

	SheetCustomPolicy = sheetDefinition{
		Name:  SheetNameCustomPolicies,
		Order: 1,
		Columns: []string{
			ColumnDisplayName, ColumnPossibleValues, ColumnDefaultValues, ColumnDescription, ColumnCategory,
			ColumnPolicyType, ColumnJustification, ColumnCostImpact,
			ColumnParameterTypes,
		},
		DynamicColumnIndex: ColumnDefaultValues,
		GetRows: func(obj interface{}) []partialRow {
			policies := obj.([]Policy)

			rows := make([]partialRow, 0, len(policies))
			for _, policy := range policies {
				row := builtInPolicyToRowValues(policy)
				row[ColumnPolicyType] = newCell("Custom")
				delete(row, ColumnResourceID)
				rows = append(rows, row)
			}
			return rows
		},
	}

	SheetASCParameters = sheetDefinition{
		Name:  SheetNameAscParameters,
		Order: 2,
		Columns: []string{
			ColumnDisplayName, ColumnPossibleValues, ColumnDefaultValues, ColumnDescription, ColumnCategory,
			ColumnPolicyType, ColumnReferenceID, ColumnJustification, ColumnCostImpact,
			ColumnParameterTypes,
		},
		DynamicColumnIndex: ColumnDefaultValues,
		GetRows: func(obj interface{}) []partialRow {
			parameters := obj.([]PolicyParameter)

			rows := make([]partialRow, 0, len(parameters))
			for _, parameter := range parameters {
				row := policyParameterToRowValues(parameter)
				rows = append(rows, row)
			}
			return rows
		},
	}
)

// SaveExcelFile saves the workbook under the target directory, properly named to avoid overwriting history files.
func SaveExcelFile(file *excelize.File, targetDir string) error {
	targetFile := getTargetFileName(targetDir)
	if _, err := os.Stat(targetFile); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		if err := os.Remove(targetFile); err != nil {
			return err
		}
	}
	return file.SaveAs(targetFile)
}

// ExportDataToExcelSheet exports input data to a sheet in the workbook, given the sheet definition and dynamic columns.
func ExportDataToExcelSheet(file *excelize.File, sheetDef sheetDefinition, data interface{}, dynamicColumns []string) error {
	sheet := newExcelSheet(file, sheetDef.Name, sheetDef.Order)

	headers, columns, err := sheetDef.BuildHeaders(dynamicColumns)
	if err != nil {
		return err
	}
	rows := []row{headers}
	for _, row := range sheetDef.GetRows(data) {
		rows = append(rows, columns.ToRowValues(row))
	}

	err = sheet.SetRows(sheetDef.Name, rows)
	if err != nil {
		return err
	}

	err = sheet.AutoFitColumnWidth(sheetDef.Name)
	if err != nil {
		return err
	}

	err = sheet.ApplyCellStyle(sheetDef.Name)
	if err != nil {
		return err
	}

	err = sheet.SetPanes(sheetDef.Name, `{
		"freeze": true,
		"split": false,
		"x_split": 0,
		"y_split": 1,
		"top_left_cell": "A2",
		"active_pane": "bottomLeft",
		"panes": [
		{
			"sqref": "A2:XFD11",
			"active_cell": "A2",
			"pane": "bottomLeft"
		}]
	}`)
	if err != nil {
		return err
	}

	return nil
}

// toPartialRows converts an input object to values for one row. It's part of the sheet definition.
type toPartialRows = func(obj interface{}) []partialRow

// Definition of sheet content which is known at compile-time.
type sheetDefinition struct {
	// The name of sheet
	Name string

	// The index of the sheet, starting from 0.
	Order int

	// Static columns, meaning they always exist.
	Columns []string

	// A function that converts the input data into rows.
	GetRows toPartialRows

	// Prefer string index instead of magic int as it is more readable.
	DynamicColumnIndex string
}

// Runtime sheet information.
type excelSheet struct {
	*excelize.File
	columns          *columns
	name             string
	cellWidth        []int
	defaultCellWidth []int
	rowCount         int
	startX           int
	startY           int
}

// all columns of a sheet, some columns can be provided at runtime.
type columns struct {
	headers            []string
	indexes            map[string]int // redundant for quick access
	dynamicColumnStart int
	dynamicColumnEnd   int
}

// Note the desired cell width may not be the same as the length of value as it can be multiple line value.
type cell struct {
	value string
	width int
}

// a row with some cells missing.
type partialRow map[string]cell

// cells of a row
type row []cell

func (c *columns) Length() int {
	return len(c.headers)
}

// ToRowValues returns a row that is aligned with the header definitions
func (c *columns) ToRowValues(values partialRow) row {
	row := make(row, len(c.headers))
	for key, value := range values {
		idx := c.indexes[key]
		row[idx] = value
	}
	return row
}

// GetDynamicColumnValues returns a column name to value mapping in which only dynamic columns are included.
// 'dynamic columns' are columns that may vary based on user inputs and cannot be hardcoded.
func (c *columns) GetDynamicColumnValues(values []string) map[string]string {
	result := make(map[string]string)
	for idx := c.dynamicColumnStart; idx < c.dynamicColumnEnd; idx++ {
		key := c.headers[idx]
		result[key] = values[idx]
	}
	return result
}

// MustGetValue returns error if value is not provided for the given column.
func (c *columns) MustGetValue(values []string, column string) (string, error) {
	v, ok := c.GetValue(values, column)
	if !ok {
		return "", fmt.Errorf("value for column '%s' does not exist", column)
	}
	return v, nil
}

// GetValue returns a flag indicating whether the value is provided for the given column.
func (c *columns) GetValue(values []string, column string) (string, bool) {
	idx, ok := c.indexes[column]
	if !ok {
		return "", false
	}
	if len(values) < idx+1 {
		return "", false
	}
	return values[idx], true
}

// BuildHeaders returns a row for header as well as the `columns` definition.
func (sd *sheetDefinition) BuildHeaders(dynamicHeaders []string) (row, *columns, error) {
	columns, err := newColumns(sd.Columns, dynamicHeaders, sd.DynamicColumnIndex)
	if err != nil {
		return nil, nil, err
	}

	values := newRow(columns.headers...)
	return values, columns, nil
}

// newExcelSheet creates a new sheet with specified name and index.
func newExcelSheet(f *excelize.File, name string, index int) excelSheet {
	if index >= 0 {
		sheetNum := len(f.GetSheetList())
		for {
			if sheetNum >= index+1 {
				break
			}
			f.NewSheet(fmt.Sprintf("Sheet%d", sheetNum+1))
			sheetNum++
		}
		oldName := f.GetSheetName(index)
		if oldName != name {
			f.SetSheetName(oldName, name)
		}
	} else if f.GetSheetIndex(name) == -1 {
		f.NewSheet(name)
	}

	return excelSheet{
		File:   f,
		name:   name,
		startX: 1,
		startY: 1,
	}
}

// SetRows sets the content of a sheet.
func (f *excelSheet) SetRows(sheetName string, rows []row) error {
	if len(rows) == 0 {
		return errors.New("rows are empty")
	}

	if f.cellWidth == nil {
		f.cellWidth = make([]int, len(rows[0]))
	}

	if f.defaultCellWidth == nil {
		f.defaultCellWidth = rows[0].Widths()
	}

	for i, row := range rows {
		y := f.startY + i
		cell, err := excelize.CoordinatesToCellName(f.startX, y)
		if err != nil {
			return err
		}
		f.updateCellWidth(row.Widths())
		values := row.Values()
		err = f.SetSheetRow(sheetName, cell, &values)
		if err != nil {
			return err
		}
	}
	f.rowCount += len(rows)
	return nil
}

// updateCellWidth updates the cell width based on length of new row values.
func (f *excelSheet) updateCellWidth(maxLens []int) {
	for i := range f.cellWidth {
		l := maxLens[i]
		if l == 0 {
			l = f.defaultCellWidth[i]
		}
		f.cellWidth[i] += l
	}
}

// AutoFitColumnWidth sets column width to proper values after all row values are known.
func (f *excelSheet) AutoFitColumnWidth(sheetName string) error {
	for i, width := range f.cellWidth {
		cellName, err := excelize.CoordinatesToCellName(f.startX+i, f.startY)
		if err != nil {
			return err
		}
		err = f.SetColWidth(sheetName, cellName[0:1], cellName[0:1], float64(width)/float64(f.rowCount)+2)
		if err != nil {
			return err
		}
	}
	return nil
}

// ApplyCellStyle applies default cell style to all cells in sheet.
func (f *excelSheet) ApplyCellStyle(sheetName string) error {
	style, err := f.NewStyle(`{
		"alignment":
		{
			"vertical": "center",
			"shrink_to_fit": true,
			"wrap_text": true
		}
	}`)
	if err != nil {
		return err
	}
	return f.SetCellStyle(sheetName, "A1", "Z999", style)
}

// create a cell with a single line value.
func newCell(v string) cell {
	return cell{
		value: v,
		width: len(v),
	}
}

// create a row with all cell containing only single line values.
func newRow(values ...string) row {
	result := make(row, 0, len(values))
	for _, value := range values {
		result = append(result, newCell(value))
	}
	return result
}

// Widths returns cell widths of a row.
func (r row) Widths() []int {
	result := make([]int, 0, len(r))
	for _, v := range r {
		result = append(result, v.width)
	}
	return result
}

// Values returns values of a row.
func (r row) Values() []string {
	result := make([]string, 0, len(r))
	for _, v := range r {
		result = append(result, v.value)
	}
	return result
}

// converts a policy parameter to row values.
func policyParameterToRowValues(parameter PolicyParameter) partialRow {
	fakeParameter := parameter
	fakeParameter.DisplayName = ""
	return map[string]cell{
		ColumnDisplayName:    newCell(parameter.DisplayName),
		ColumnPossibleValues: newCell(formatParameterPossibleValues(fakeParameter)),
		ColumnDefaultValues:  newCell(formatParameterDefaultValues(fakeParameter)),
		ColumnDescription:    newCell(parameter.Description),
		ColumnCategory:       newCell("Security Center"),
		ColumnPolicyType:     newCell("Builtin"),
		ColumnReferenceID:    newCell(parameter.InternalName),
		ColumnJustification:  newCell(parameter.Justification),
		ColumnCostImpact:     newCell(parameter.CostImpact),
		ColumnParameterTypes: newCell(formatParameterTypes(fakeParameter)),
	}
}

// converts a builtin policy to row values.
func builtInPolicyToRowValues(policy Policy) partialRow {
	parameters := getParametersOfPolicyForExport(policy)
	return map[string]cell{
		ColumnDisplayName:    newCell(policy.DisplayName),
		ColumnPossibleValues: formatParameters(parameters, formatParameterPossibleValues),
		ColumnDefaultValues:  formatParameters(parameters, formatParameterDefaultValues),
		ColumnDescription:    newCell(policy.Description),
		ColumnCategory:       newCell(policy.Category),
		ColumnPolicyType:     newCell("Builtin"),
		ColumnResourceID:     newCell(policy.ResourceID),
		ColumnJustification:  newCell(policy.Justification),
		ColumnCostImpact:     newCell(policy.CostImpact),
		ColumnParameterTypes: formatParameters(policy.Parameters, formatParameterTypes),
	}
}

// build cell value for parameters.
func formatParameters(parameters []PolicyParameter, toString func(PolicyParameter) string) cell {
	values := make([]string, 0, len(parameters))
	maxLen := 0
	for _, param := range parameters {
		value := toString(param)
		if l := len(value); maxLen < l {
			maxLen = l
		}
		values = append(values, value)
	}
	return cell{
		value: strings.Join(values, "\n"),
		width: maxLen,
	}
}

// build cell value of types of parameters.
func formatParameterTypes(param PolicyParameter) string {
	return fmt.Sprintf("%s: %s", param.InternalName, param.Type)
}

// build cell value of possible values of parameters.
func formatParameterPossibleValues(param PolicyParameter) string {
	allowedValues := make([]string, 0, len(param.AllowedValues))
	for _, allowedValue := range param.AllowedValues {
		allowedValues = append(allowedValues, cast.ToString(allowedValue))
	}
	if len(allowedValues) == 0 {
		allowedValues = append(allowedValues, fmt.Sprintf("<%s>", param.Type))
	}
	return fmt.Sprintf("%s: %s", param.InternalName, strings.Join(allowedValues, ";"))
}

// build cell value of default values of parameters.
func formatParameterDefaultValues(param PolicyParameter) string {
	var defaultValue string
	if param.DefaultValue != nil {
		defaultValue = cast.ToString(param.DefaultValue)
	} else {
		defaultValue = "<>"
	}
	return fmt.Sprintf("%s: %s", param.InternalName, defaultValue)
}

// Build column definitions with all columns provided.
func newColumns(staticHeaders []string, dynamicHeaders []string, dynamicHeaderIndex string) (*columns, error) {
	headers := make([]string, 0, len(staticHeaders)+len(dynamicHeaders)+1)
	insertAt := -1
	for i, header := range staticHeaders {
		if header == dynamicHeaderIndex {
			insertAt = i
			break
		}
	}
	if insertAt == -1 {
		return nil, fmt.Errorf("the index '%s' for management group is invalid", dynamicHeaderIndex)
	}
	insertAt++
	headers = append(headers, staticHeaders[0:insertAt]...)
	headers = append(headers, dynamicHeaders...)
	headers = append(headers, staticHeaders[insertAt:]...)

	headerIndexes := make(map[string]int)
	for i, header := range headers {
		headerIndexes[header] = i
	}

	columns := columns{
		headers:            headers,
		indexes:            headerIndexes,
		dynamicColumnStart: insertAt,
		dynamicColumnEnd:   insertAt + len(dynamicHeaders),
	}
	return &columns, nil
}

func getParametersOfPolicyForExport(policy Policy) []PolicyParameter {
	parameters := policy.Parameters
	var effectParam *PolicyParameter
	for i := range parameters {
		if parameters[i].InternalName == "effect" {
			effectParam = &parameters[i]
		}
	}
	if effectParam != nil {
		if effectParam.DefaultValue == nil {
			effectParam.DefaultValue = policy.Effect
		}
	} else {
		parameters = append(parameters, PolicyParameter{
			InternalName: "*effect",
			Type:         "string",
			DefaultValue: policy.Effect,
		})
	}
	return parameters
}

func getTargetFileName(targetDir string) string {
	return filepath.Join(
		targetDir,
		fmt.Sprintf(
			"Azure Cloud Foundation - Baseline Policies - %s.xlsx",
			time.Now().Format("20060102"),
		),
	)
}
