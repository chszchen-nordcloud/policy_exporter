package exporter

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
	"testing"
)

func TestExportBuiltInPoliciesToExcel(t *testing.T) {
	if SkipTest() {
		return
	}

	az, err := NewAzureAPI("")
	assert.NoError(t, err)

	policies, err := az.ListBuiltInPolicyByManagementGroup(context.Background(), "Sandbox")
	assert.NoError(t, err)

	managementGroups := []string{"MANAGEMENT", "PRODUCTION", "NON-PROD", "SANDBOX"}
	f := excelize.NewFile()
	err = ExportDataToExcelSheet(f, SheetBuiltInPolicy, policies, managementGroups)
	assert.NoError(t, err)

	err = SaveExcelFile(f, TestResourceDir())
	assert.NoError(t, err)
}

func TestExportCustomPoliciesToExcel(t *testing.T) {
	if SkipTest() {
		return
	}

	policies, err := ReadCustomPoliciesFromLocalRepository("/tmp/ACF3.Modular")
	assert.NoError(t, err)

	managementGroups := []string{"MANAGEMENT", "PRODUCTION", "NON-PROD", "SANDBOX"}
	f := excelize.NewFile()
	err = ExportDataToExcelSheet(f, SheetCustomPolicy, policies, managementGroups)
	assert.NoError(t, err)

	err = SaveExcelFile(f, TestResourceDir())
	assert.NoError(t, err)
}

func TestExportASCPolicyParameters(t *testing.T) {
	if SkipTest() {
		return
	}

	az, err := NewAzureAPI("")
	assert.NoError(t, err)

	params, err := az.GetPolicySetParameters(context.Background(), "1f3afdf9-d0c9-4c3d-847f-89da613e70a8")
	assert.NoError(t, err)

	subscriptions := []string{"PLATFORM", "LANDING ZONE", "SANDBOX"}
	f := excelize.NewFile()
	err = ExportDataToExcelSheet(f, SheetASCParameters, params, subscriptions)
	assert.NoError(t, err)

	err = SaveExcelFile(f, TestResourceDir())
	assert.NoError(t, err)
}

func getAdhocPolicies() []Policy {
	return []Policy{
		{
			DisplayName: "Microsoft Managed Control 1599 - Developer Configuration Management",
			ResourceID:  "/providers/Microsoft.Authorization/policyDefinitions/0004bbf0-5099-4179-869e-e9ffe5fb0945",
			Description: "Audit virtual machines which do not have disaster recovery configured. To learn more about disaster recovery, visit https://aka.ms/asr-doc.",
			ManagementGroups: map[string]Attachment{
				"MANAGEMENT": {"Effect": "Audit"},
				"PRODUCTION": {"Effect": "Audit"},
				"NON-PROD":   {"Effect": "Audit"},
				"SANDBOX":    {"Effect": "Audit"},
			},
		},
	}
}
