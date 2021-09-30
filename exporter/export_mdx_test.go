package exporter

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExportBuiltInPolicyDoc(t *testing.T) {
	if SkipTest() {
		return
	}
	policies := []Policy{
		{
			DisplayName: "Microsoft Managed Control 1599 - Developer Configuration Management",
			ResourceID:  "/providers/Microsoft.Authorization/policyDefinitions/0004bbf0-5099-4179-869e-e9ffe5fb0945",
			Description: "Audit virtual machines which do not have disaster recovery configured. To learn more about disaster recovery, visit https://aka.ms/asr-doc.",
			ManagementGroups: map[string]Attachment{
				"MANAGEMENT": {Effect: "Audit"},
				"PRODUCTION": {Effect: "Audit"},
				"NON-PROD":   {Effect: "Audit"},
				"SANDBOX":    {Effect: "Audit"},
			},
		},
	}
	managementGroups := []string{"MANAGEMENT", "PRODUCTION", "NON-PROD", "SANDBOX"}
	err := ExportBuiltInPolicyDoc(managementGroups, policies, TestResourceDir())
	assert.NoError(t, err)
}

func TestExportASCPolicyDoc(t *testing.T) {
	if SkipTest() {
		return
	}
	params := []PolicyParameter{
		{
			InternalName: "A managed identity should be used in your API App",
			DisplayName:  "A managed identity should be used in your API App",
			ManagementGroups: map[string]interface{}{
				"MANAGEMENT": "AuditIfNotExists",
				"PRODUCTION": "AuditIfNotExists",
				"NON-PROD":   "AuditIfNotExists",
				"SANDBOX":    "AuditIfNotExists",
			},
			Description:   "Use a managed identity for enhanced authentication security",
			Justification: "Managed Identity should be used to allow centrilez managemnt and increase security of API apps",
		},
	}
	managementGroups := []string{"MANAGEMENT", "PRODUCTION", "NON-PROD", "SANDBOX"}
	err := ExportASCPolicyDoc(managementGroups, params, TestResourceDir())
	assert.NoError(t, err)
}
