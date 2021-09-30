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

	headers, columns, err := sheetDef.GetHeaders(dynamicColumns)
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
	Name               string
	Order              int
	Columns            []string
	GetRows            toPartialRows
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

type columns struct {
	Headers            []string
	Indexes            map[string]int
	DynamicColumnStart int
	DynamicColumnEnd   int
}

// Note the desired length may not be the same as the length of value as it can be multiple line value.
type cell struct {
	value  string
	maxLen int
}

type partialRow map[string]cell

type row []cell

func (c *columns) Length() int {
	return len(c.Headers)
}

func (c *columns) ToRowValues(values partialRow) row {
	row := make(row, len(c.Headers))
	for key, value := range values {
		idx := c.Indexes[key]
		row[idx] = value
	}
	return row
}

func (c *columns) GetDynamicColumnValues(values []string) map[string]string {
	result := make(map[string]string)
	for idx := c.DynamicColumnStart; idx < c.DynamicColumnEnd; idx++ {
		key := c.Headers[idx]
		result[key] = values[idx]
	}
	return result
}

func (c *columns) MustGetValue(values []string, column string) (string, error) {
	v, ok := c.GetValue(values, column)
	if !ok {
		return "", fmt.Errorf("value for column '%s' does not exist", column)
	}
	return v, nil
}

func (c *columns) GetValue(values []string, column string) (string, bool) {
	idx, ok := c.Indexes[column]
	if !ok {
		return "", false
	}
	if len(values) < idx+1 {
		return "", false
	}
	return values[idx], true
}

// GetHeaders returns headers as row values.
func (sd *sheetDefinition) GetHeaders(dynamicHeaders []string) (row, *columns, error) {
	columns, err := newColumns(sd.Columns, dynamicHeaders, sd.DynamicColumnIndex)
	if err != nil {
		return nil, nil, err
	}

	values := newCellValues(columns.Headers...)
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
		f.defaultCellWidth = rows[0].MaxLens()
	}

	for i, row := range rows {
		y := f.startY + i
		cell, err := excelize.CoordinatesToCellName(f.startX, y)
		if err != nil {
			return err
		}
		f.updateCellWidth(row.MaxLens())
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

func newCell(v string) cell {
	return cell{
		value:  v,
		maxLen: len(v),
	}
}

func newCellValues(values ...string) []cell {
	result := make([]cell, 0, len(values))
	for _, value := range values {
		result = append(result, newCell(value))
	}
	return result
}

func (r row) MaxLens() []int {
	result := make([]int, 0, len(r))
	for _, v := range r {
		result = append(result, v.maxLen)
	}
	return result
}

func (r row) Values() []string {
	result := make([]string, 0, len(r))
	for _, v := range r {
		result = append(result, v.value)
	}
	return result
}

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

func builtInPolicyToRowValues(policy Policy) partialRow {
	return map[string]cell{
		ColumnDisplayName:    newCell(policy.DisplayName),
		ColumnPossibleValues: formatParameters(policy.Parameters, formatParameterPossibleValues),
		ColumnDefaultValues:  formatParameters(policy.Parameters, formatParameterDefaultValues),
		ColumnDescription:    newCell(policy.Description),
		ColumnCategory:       newCell(policy.Category),
		ColumnPolicyType:     newCell("Builtin"),
		ColumnResourceID:     newCell(policy.ResourceID),
		ColumnJustification:  newCell(policy.Justification),
		ColumnCostImpact:     newCell(policy.CostImpact),
		ColumnParameterTypes: formatParameters(policy.Parameters, formatParameterTypes),
	}
}

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
		value:  strings.Join(values, "\n"),
		maxLen: maxLen,
	}
}

func formatParameterTypes(param PolicyParameter) string {
	return fmt.Sprintf("%s: %s", param.InternalName, param.Type)
}

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

func formatParameterDefaultValues(param PolicyParameter) string {
	var defaultValue string
	if param.DefaultValue != nil {
		defaultValue = cast.ToString(param.DefaultValue)
	} else {
		defaultValue = "<>"
	}
	return fmt.Sprintf("%s: %s", param.InternalName, defaultValue)
}

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
		Headers:            headers,
		Indexes:            headerIndexes,
		DynamicColumnStart: insertAt,
		DynamicColumnEnd:   insertAt + len(dynamicHeaders),
	}
	return &columns, nil
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
