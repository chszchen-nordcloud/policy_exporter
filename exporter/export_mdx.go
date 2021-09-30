package exporter

import (
	"github.com/fatih/color"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	TMPL_BUILTIN_POLICY_TABLE = `
The content is update at {{.Timestamp}}

ResourceId | DisplayName {{range .ManagementGroups}}| {{.}}{{end}}| Description | Justification 
----------- | ----------- {{range .ManagementGroups}}| ----------- {{end}}| ----------- | ----------- 
{{range $policy := .Policies}}{{.ResourceID}}      | {{.DisplayName}} {{range $groupName := $.ManagementGroups}}| {{with index $policy.ManagementGroups $groupName}}{{.Effect}}{{end}} {{end}} | {{.Description}} | {{.Justification }}
{{end}}
`
	TMPL_CUSTOM_POLICY_TABLE = `
The content is update at {{.Timestamp}}

ResourceId | DisplayName {{range .ManagementGroups}}| {{.}}{{end}}| Description | Justification |
----------- | ----------- {{range .ManagementGroups}}| ----------- {{end}}| ----------- | ----------- |
{{range $policy := .Policies}}{{.ResourceID}}      | {{.DisplayName}} {{range $groupName := $.ManagementGroups}}| {{with index $policy.ManagementGroups $groupName}}{{.Effect}}{{end}} {{end}} | {{.Description}} | {{.Justification }}
{{end}}
`
	TMPL_ASC_POLICY_TABLE = `
The content is update at {{.Timestamp}}

Internal Name | Policy Definition {{range .ManagementGroups}}| {{.}}{{end}}| Description | Justification | Cost Impact
----------- | ----------- {{range .ManagementGroups}}| ----------- {{end}}| ----------- | ----------- | ----------- 
{{range $param := .Parameters }}{{.DisplayName}}      | {{.DisplayName}} {{range $groupName := $.ManagementGroups}}| {{with index $param.ManagementGroups $groupName}}{{.}}{{end}} {{end}} | {{.Description}} | {{.Justification }} | {{.CostImpact}}
{{end}}
`
)

type DocumentInput struct {
	ManagementGroups           []string
	BuiltInPolicyLastUpdatedAt string
	BuiltInPolicies            []Policy
	CustomPolicyLastUpdatedAt  string
	CustomPolicies             []Policy
	ASCPolicyLastUpdatedAt     string
	ASCPolicySetParameters     []PolicyParameter
}

func ExportBuiltInPolicyDoc(managementGroups []string, policies []Policy, targetDir string) error {
	fd, err := os.Create(filepath.Join(targetDir, "BuiltInPolicies.mdx"))
	if err != nil {
		return err
	}
	tmpl := template.Must(template.New("Policy").Parse(TMPL_BUILTIN_POLICY_TABLE))
	color.Green("Write to %s under %s", fd.Name(), targetDir)

	policiesToExport := make([]Policy, 0, len(policies))
	for _, policy := range policies {
		policy.DisplayName = escapeStringForMarkdown(policy.DisplayName)
		policiesToExport = append(policiesToExport, policy)
	}

	return tmpl.Execute(fd, map[string]interface{}{
		"ManagementGroups": managementGroups,
		"Policies":         policiesToExport,
		"Timestamp":        currentTimeStamp(),
	})
}

func escapeStringForMarkdown(s string) string {
	return strings.ReplaceAll(s, "|", `\|`)
}

func ExportASCPolicyDoc(managementGroups []string, parameters []PolicyParameter, targetDir string) error {
	fd, err := os.Create(filepath.Join(targetDir, "ASCPolicySetParameters.mdx"))
	if err != nil {
		return err
	}
	tmpl := template.Must(template.New("Parameter").Parse(TMPL_ASC_POLICY_TABLE))
	color.Green("Write to %s under %s", fd.Name(), targetDir)
	return tmpl.Execute(fd, map[string]interface{}{
		"ManagementGroups": managementGroups,
		"Parameters":       parameters,
		"Timestamp":        currentTimeStamp(),
	})
}

func currentTimeStamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
