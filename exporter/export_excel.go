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

// Columns are used both for column names in Excel file and indexes into the row values.
// We prefer string index over hardcoded magic integers.
const (
	ColumnDisplayName          = "DisplayName"
	ColumnPossibleValues       = "Parameters: Possible values"
	ColumnDefaultValues        = "Default values"
	ColumnDescription          = "Description"
	ColumnCategory             = "Category"
	ColumnPolicyType           = "Policy Type"
	ColumnResourceID           = "ResourceID"
	ColumnJustification        = "Justification"
	ColumnCostImpact           = "Cost Impact"
	ColumnReferenceID          = "Reference ID"
	ColumnBelongingInitiatives = "Belonging Initiatives"
	ColumnParameterTypes       = "Parameter Types"
	ColumnRecommendation       = "Baseline Recommendation"
)

var (
	SheetBuiltInPolicy = sheetDefinition{
		Name:  SheetNameBuiltinPolicies,
		Order: 0,
		Columns: []string{
			ColumnDisplayName, ColumnPossibleValues, ColumnDefaultValues, ColumnDescription, ColumnCategory,
			ColumnPolicyType, ColumnResourceID, ColumnJustification, ColumnCostImpact,
			ColumnBelongingInitiatives, ColumnParameterTypes, ColumnRecommendation,
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
			ColumnPolicyType, ColumnJustification, ColumnCostImpact, ColumnRecommendation,
			ColumnParameterTypes,
		},
		DynamicColumnIndex: ColumnDefaultValues,
		GetRows: func(obj interface{}) []partialRow {
			policies := obj.([]Policy)

			rows := make([]partialRow, 0, len(policies))
			for _, policy := range policies {
				row := builtInPolicyToRowValues(policy)
				row[ColumnPolicyType] = newCell("Custom")
				delete(row, ColumnBelongingInitiatives)
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
	targetFile := getTargetIntermediateExcelFileName(targetDir)
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
	sheet, err := newExcelSheet(file, sheetDef, dynamicColumns)
	if err != nil {
		return err
	}

	partialRows := sheetDef.GetRows(data)
	err = sheet.SetRows(partialRows)
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

// sheetDefinition contains definition of sheet which is known at compile-time.
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

// excelSheet contains runtime sheet information.
type excelSheet struct {
	*excelize.File
	name string

	// columns is runtime column definition
	columns *columns

	// cellWidth keeps the cell width for each column
	cellWidth []int

	// defaultCellWidth serves as a default value for cell width in case the calculated width does not make sense.
	// As the width is dynamically updated based on cell values, it can be very small if only a few cells have values
	// for a column.
	defaultCellWidth []int

	rowCount int

	// The column index at which the content begins.
	startX int

	// the row index at which the content begins.
	startY int
}

// columns are all columns of a sheet, some columns can be provided at runtime.
type columns struct {
	headers            []string
	indexes            map[string]int // redundant for quick access
	dynamicColumnStart int
	dynamicColumnEnd   int
}

// cell contains information for rendering a cell.
// Note the desired cell width may not be the same as the length of value as it can be multiple line value.
type cell struct {
	value string
	width int
}

// partialRow contains cells whose values are not empty.
type partialRow map[string]cell

// row contains cells of all columns, including empty cells if needed.
type row []cell

func (c *columns) Length() int {
	return len(c.headers)
}

// ToRowValues converts a partialRow to a row.
func (c *columns) ToRowValues(values partialRow) row {
	row := make(row, len(c.headers))
	for key, value := range values {
		idx, ok := c.indexes[key]
		if ok {
			row[idx] = value
		} else {
			fmt.Printf("column %s from row value is not found in sheet definition\n", key)
		}
	}
	return row
}

// newExcelSheet creates a new sheet with specified name and index.
func newExcelSheet(f *excelize.File, definition sheetDefinition, dynamicColumns []string) (*excelSheet, error) {
	sheetIndex := definition.Order
	sheetName := definition.Name
	if definition.Order >= 0 {
		sheetNum := len(f.GetSheetList())
		for {
			if sheetNum >= sheetIndex+1 {
				break
			}
			f.NewSheet(fmt.Sprintf("Sheet%d", sheetNum+1))
			sheetNum++
		}
		oldName := f.GetSheetName(sheetIndex)
		if oldName != sheetName {
			f.SetSheetName(oldName, sheetName)
		}
	} else if f.GetSheetIndex(sheetName) == -1 {
		f.NewSheet(sheetName)
	}

	cols, err := newColumns(definition.Columns, dynamicColumns, definition.DynamicColumnIndex)
	if err != nil {
		return nil, err
	}

	sheet := excelSheet{
		File:    f,
		name:    sheetName,
		columns: cols,
		startX:  1,
		startY:  1,
	}
	return &sheet, nil
}

// GetRows is used to read excel contents. Currently used for testing purpose.
func (f *excelSheet) GetRows() ([]namedCells, error) {
	rows, err := f.File.GetRows(f.name)
	if err != nil {
		return nil, err
	}

	if f.columns.Length() != len(rows[0]) {
		return nil, fmt.Errorf("columns length %d does not match values length %d in sheet %s", f.columns.Length(), len(rows[0]), f.name)
	}

	result := make([]namedCells, 0, len(rows)-1)
	for _, row := range rows[1:] {
		result = append(result, namedCells{
			columns: f.columns,
			values:  row,
		})
	}
	return result, nil
}

// SetRows sets the content of a sheet.
func (f *excelSheet) SetRows(partialRows []partialRow) error {
	if len(partialRows) == 0 {
		return errors.New("rows are empty")
	}

	headerRow := newRow(f.columns.headers...)
	if f.cellWidth == nil {
		f.cellWidth = make([]int, len(headerRow))
	}
	if f.defaultCellWidth == nil {
		f.defaultCellWidth = headerRow.Widths()
	}

	rows := []row{headerRow}
	for _, row := range partialRows {
		rows = append(rows, f.columns.ToRowValues(row))
	}

	for i, row := range rows {
		y := f.startY + i
		cell, err := excelize.CoordinatesToCellName(f.startX, y)
		if err != nil {
			return err
		}
		f.updateCellWidth(row.Widths())
		values := row.Values()
		err = f.SetSheetRow(f.name, cell, &values)
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
		widthToUse := float64(width)/float64(f.rowCount) + 2
		if widthToUse > 255.0 {
			widthToUse = 255.0
		}
		err = f.SetColWidth(sheetName, cellName[0:1], cellName[0:1], widthToUse)
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

// policyParameterToRowValues converts a policy parameter to row values.
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

// builtInPolicyToRowValues converts a builtin policy to row values.
func builtInPolicyToRowValues(policy Policy) partialRow {
	recommendValue := "No"
	if policy.Recommend {
		recommendValue = "Yes"
	}
	return map[string]cell{
		ColumnDisplayName:          newCell(policy.DisplayName),
		ColumnPossibleValues:       formatParameters(policy.GetParametersForExport(), formatParameterPossibleValues),
		ColumnDefaultValues:        formatParameters(policy.Parameters, formatParameterDefaultValues),
		ColumnDescription:          newCell(policy.Description),
		ColumnCategory:             newCell(policy.Category),
		ColumnPolicyType:           newCell("Builtin"),
		ColumnResourceID:           newCell(policy.ResourceID),
		ColumnJustification:        newCell(policy.Justification),
		ColumnCostImpact:           newCell(policy.CostImpact),
		ColumnBelongingInitiatives: formatValues(policy.InitiativeIDs),
		ColumnParameterTypes:       formatParameters(policy.Parameters, formatParameterTypes),
		ColumnRecommendation:       newCell(fmt.Sprintf("%-8s", recommendValue)),
	}
}

// formatValues formats multiple values so that it fit a cell.
func formatValues(values []string) cell {
	maxLen := 0
	for _, value := range values {
		if l := len(value); maxLen < l {
			maxLen = l
		}
	}
	return cell{
		value: strings.Join(values, "\n"),
		width: maxLen,
	}
}

// formatParameters build cell value for parameters.
func formatParameters(parameters []PolicyParameter, toString func(PolicyParameter) string) cell {
	values := make([]string, 0, len(parameters))
	for _, param := range parameters {
		value := toString(param)
		values = append(values, value)
	}
	return formatValues(values)
}

// formatParameterTypes builds cell value of types of parameters.
func formatParameterTypes(param PolicyParameter) string {
	return fmt.Sprintf("%s: %s", param.InternalName, param.Type)
}

// formatParameterPossibleValues builds cell value of possible values of parameters.
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

// formatParameterDefaultValues builds cell value of default values of parameters.
func formatParameterDefaultValues(param PolicyParameter) string {
	var defaultValue string
	if param.DefaultValue != nil {
		defaultValue = cast.ToString(param.DefaultValue)
	} else {
		defaultValue = "<>"
	}
	return fmt.Sprintf("%s: %s", param.InternalName, defaultValue)
}

// newColumns builds column definitions with all columns provided.
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

func getTargetIntermediateExcelFileName(targetDir string) string {
	return filepath.Join(
		targetDir,
		fmt.Sprintf(
			"Azure Cloud Foundation - Baseline Policies - %s.xlsx",
			time.Now().Format("20060102"),
		),
	)
}
