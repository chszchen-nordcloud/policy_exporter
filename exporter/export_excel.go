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

func ExportDataToExcelSheet(file *excelize.File, sheetDef sheetDefinition, data interface{}, dynamicColumns []string) error {
	sheet := newExcelSheet(file, sheetDef.Name, sheetDef.Order)

	headers, err := sheetDef.GetHeaders(dynamicColumns)
	if err != nil {
		return err
	}
	rows := []rowValues{headers}
	rows = append(rows, sheetDef.GetRows(data, dynamicColumns)...)

	err = sheet.AddRows(sheetDef.Name, rows)
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

var (
	SheetBuiltInPolicy = sheetDefinition{
		Name:  SHEET_NAME_BUILTIN_POLICIES,
		Order: 0,
		Columns: []string{
			"DisplayName", "Parameters: Possible values", "Default values", "Description", "Category", "Policy Type", "ResourceId",
			"Justification", "Cost Impact",
		},
		DynamicColumnIndex: "Default values",
		GetRows: func(obj interface{}, managementGroups []string) []rowValues {
			policies := obj.([]Policy)

			rows := make([]rowValues, 0, len(policies))
			for _, policy := range policies {
				row := builtInPolicyToRowValues(policy, managementGroups)
				rows = append(rows, row)
			}
			return rows
		},
	}

	SheetCustomPolicy = sheetDefinition{
		Name:  SHEET_NAME_CUSTOM_POLICIES,
		Order: 1,
		Columns: []string{
			"DisplayName", "Parameters: Possible values", "Default values", "Description", "Category", "PolicyType",
			"Justification", "Cost Impact",
		},
		DynamicColumnIndex: "Default values",
		GetRows: func(obj interface{}, managementGroups []string) []rowValues {
			policies := obj.([]Policy)

			rows := make([]rowValues, 0, len(policies))
			for _, policy := range policies {
				row := builtInPolicyToRowValues(policy, managementGroups)
				row[(len(row) - 4)] = newCellValue("Custom")
				row[len(row)-3] = newCellValue("")
				rows = append(rows, row[0:len(row)-1])
			}
			return rows
		},
	}

	SheetASCParameters = sheetDefinition{
		Name:  SHEET_NAME_ASC_PARAMETERS,
		Order: 2,
		Columns: []string{
			"DisplayName", "Parameters: Possible values", "Default values", "Description", "Category", "PolicyType",
			"Reference ID", "Justification", "Cost Impact",
		},
		DynamicColumnIndex: "Default values",
		GetRows: func(obj interface{}, subscriptions []string) []rowValues {
			parameters := obj.([]PolicyParameter)

			rows := make([]rowValues, 0, len(parameters))
			for _, parameter := range parameters {
				row := policyParameterToRowValues(parameter, subscriptions)
				rows = append(rows, row)
			}
			return rows
		},
	}
)

type toRows = func(obj interface{}, dynamicColumns []string) []rowValues

type sheetDefinition struct {
	Name               string
	Order              int
	Columns            []string
	GetRows            toRows
	DynamicColumnIndex string
}

type excelSheet struct {
	*excelize.File
	name             string
	cellWidth        []int
	defaultCellWidth []int
	rowCount         int
	startX           int
	startY           int
}

type cellValue struct {
	value  string
	maxLen int
}

type rowValues []cellValue

func (sd *sheetDefinition) GetHeaders(dynamicHeaders []string) (rowValues, error) {
	headers, err := prepareHeaders(sd.Columns, dynamicHeaders, sd.DynamicColumnIndex)
	if err != nil {
		return nil, err
	}
	values := newCellValues(headers...)
	return values, nil
}

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

func (f *excelSheet) AddRows(sheetName string, rows []rowValues) error {
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
		f.UpdateCellWidth(row.MaxLens())
		values := row.Values()
		err = f.SetSheetRow(sheetName, cell, &values)
		if err != nil {
			return err
		}
	}
	f.rowCount += len(rows)
	return nil
}

func (f *excelSheet) UpdateCellWidth(maxLens []int) {
	for i := range f.cellWidth {
		l := maxLens[i]
		if l == 0 {
			l = f.defaultCellWidth[i]
		}
		f.cellWidth[i] += l
	}
}

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

func newCellValue(v string) cellValue {
	return cellValue{
		value:  v,
		maxLen: len(v),
	}
}

func newCellValues(values ...string) []cellValue {
	result := make([]cellValue, 0, len(values))
	for _, value := range values {
		result = append(result, newCellValue(value))
	}
	return result
}

func (r rowValues) MaxLens() []int {
	result := make([]int, 0, len(r))
	for _, v := range r {
		result = append(result, v.maxLen)
	}
	return result
}

func (r rowValues) Values() []string {
	result := make([]string, 0, len(r))
	for _, v := range r {
		result = append(result, v.value)
	}
	return result
}

func policyParameterToRowValues(parameter PolicyParameter, subscriptions []string) rowValues {
	fakeParameter := parameter
	fakeParameter.DisplayName = ""

	values := newCellValues(
		parameter.DisplayName,
		formatParameterPossibleValues(fakeParameter),
		formatParameterDefaultValues(fakeParameter),
	)
	values = append(values, make([]cellValue, len(subscriptions))...)
	values = append(values, newCellValues(
		parameter.Description,
		"Security Center",
		"BuiltIn",
		parameter.InternalName,
		parameter.Justification,
		parameter.CostImpact,
	)...)
	return values
}

func builtInPolicyToRowValues(policy Policy, managementGroups []string) rowValues {
	values := []cellValue{
		newCellValue(policy.DisplayName),
		formatParameters(policy.Parameters, formatParameterPossibleValues),
		formatParameters(policy.Parameters, formatParameterDefaultValues),
	}
	values = append(values, make([]cellValue, len(managementGroups))...)
	values = append(values, newCellValues(
		policy.Description,
		policy.Category,
		"BuiltIn",
		policy.ResourceID,
		policy.Justification,
		policy.CostImpact,
	)...)
	return values
}

func formatParameters(parameters []PolicyParameter, toString func(PolicyParameter) string) cellValue {
	values := make([]string, 0, len(parameters))
	maxLen := 0
	for _, param := range parameters {
		value := toString(param)
		if l := len(value); maxLen < l {
			maxLen = l
		}
		values = append(values, value)
	}
	return cellValue{
		value:  strings.Join(values, "\n"),
		maxLen: maxLen,
	}
}

func formatParameterPossibleValues(param PolicyParameter) string {
	allowedValues := make([]string, 0, len(param.AllowedValues))
	for _, allowedValue := range param.AllowedValues {
		allowedValues = append(allowedValues, cast.ToString(allowedValue))
	}
	if len(allowedValues) == 0 {
		allowedValues = append(allowedValues, fmt.Sprintf("<%s>", param.Type))
	}
	return fmt.Sprintf("%s: %s", param.DisplayName, strings.Join(allowedValues, ";"))
}

func formatParameterDefaultValues(param PolicyParameter) string {
	var defaultValue string
	if param.DefaultValue != nil {
		defaultValue = cast.ToString(param.DefaultValue)
	} else {
		defaultValue = "<>"
	}
	return fmt.Sprintf("%s: %s", param.DisplayName, defaultValue)
}

func prepareHeaders(staticHeaders []string, dynamicHeaders []string, dynamicHeaderIndex string) ([]string, error) {
	result := make([]string, 0, len(staticHeaders)+len(dynamicHeaders)+1)
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
	result = append(result, staticHeaders[0:insertAt]...)
	result = append(result, dynamicHeaders...)
	result = append(result, staticHeaders[insertAt:]...)
	return result, nil
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
