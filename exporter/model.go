package exporter

import (
	"context"
	"strings"
)

type PolicyDefinitionProvider struct {
	BuiltInPolicyReader         PolicyReader
	CustomPolicyReader          PolicyReader
	ASCPolicySetParameterReader PolicyParameterReader
}

type PolicyDefinitionExporter struct {
	// Will be used instead of other policy exporters if present
	PolicyExporter PolicyExporter

	BuiltInPolicyExporter      PolicyExporter
	CustomPolicyExporter       PolicyExporter
	PolicySetParameterExporter PolicyParameterExporter
}

type PolicyReader = func(ctx context.Context) ([]Policy, error)

type PolicyParameterReader = func(ctx context.Context) ([]PolicyParameter, error)

// PolicyExporter should be responsible for making its exported file names unique.
type PolicyExporter = func(policies []Policy, targetDir string) error

type PolicyParameterExporter = func(parameters []PolicyParameter, targetDir string) error

type SortPoliciesByDisplayName []Policy

type SortPolicyParametersByInternalName []PolicyParameter

// Category is used in Azure LandingZone JSON parameters files.
type Category struct {
	Name     string   `json:"name"`
	Policies []Policy `json:"policies"`
}

type Policy struct {
	Category         string                `json:"-" yaml:"Category"`
	DisplayName      string                `json:"name" yaml:"DisplayName"`
	ResourceID       string                `json:"-" yaml:"ResourceID"`
	Justification    string                `json:"-" yaml:"Justification"`
	CostImpact       string                `json:"-" yaml:"CostImpact"`
	Description      string                `json:"-" yaml:"Description"`
	Parameters       []PolicyParameter     `json:"-" yaml:"-"`
	ManagementGroups map[string]Attachment `json:"managementGroups" yaml:"ManagementGroups"`
	Required         bool                  `json:"-" yaml:"-"`
}

// Attachment attaches a policy to a management group.
type Attachment struct {
	Enabled    bool                   `json:"enabled" yaml:"Enabled"`
	Parameters map[string]interface{} `json:"parameters" yaml:"Parameters"`
	Location   string                 `json:"location" yaml:"Location"`
	Effect     string                 `json:"-" yaml:"Effect"`
}

type PolicyParameter struct {
	InternalName     string
	Type             string
	DisplayName      string
	Description      string
	DefaultValue     interface{}
	Justification    string
	CostImpact       string
	AllowedValues    []interface{}
	ManagementGroups map[string]interface{}
	Required         bool `json:"-" yaml:"-"`
}

// PolicyParameterValue is used in Azure LandingZone JSON parameter file.
type PolicyParameterValue struct {
	Value interface{} `json:"value"`
}

// Merge merges two policy definitions into one.
func (p *Policy) Merge(other Policy) {
	if p.DisplayName == "" {
		p.DisplayName = other.DisplayName
	}
	if p.ResourceID == "" {
		p.ResourceID = other.ResourceID
	}
	if p.Justification == "" {
		p.Justification = other.Justification
	}
	if p.Description == "" {
		p.Description = other.Justification
	}
	if p.Category == "" {
		p.Category = other.Category
	}
	if p.Justification == "" {
		p.Justification = other.Justification
	}
	if p.CostImpact == "" {
		p.CostImpact = other.CostImpact
	}
	p.Required = p.Required || other.Required
	if p.ManagementGroups == nil {
		p.ManagementGroups = make(map[string]Attachment)
	}
	managementGroups := &p.ManagementGroups
	for k, v := range other.ManagementGroups {
		if existing, ok := (*managementGroups)[k]; ok {
			existing.Merge(v)
			(*managementGroups)[k] = existing
		} else {
			(*managementGroups)[k] = v
		}
	}
}

func (a *Attachment) Merge(other Attachment) {
	if other.Enabled {
		a.Enabled = other.Enabled
	}
	if a.Location == "" {
		a.Location = other.Location
	}
	if a.Effect == "" {
		a.Effect = other.Effect
	}
	for k, v := range other.Parameters {
		a.Parameters[k] = v
	}
}

func (p *PolicyParameter) Merge(other PolicyParameter) {
	if p.InternalName == "" {
		p.InternalName = other.InternalName
	}
	if p.DisplayName == "" {
		p.DisplayName = other.DisplayName
	}
	if p.Description == "" {
		p.Description = other.Description
	}
	if p.DefaultValue == "" {
		p.DefaultValue = other.DefaultValue
	}
	if p.Justification == "" {
		p.Justification = other.Justification
	}
	if p.CostImpact == "" {
		p.CostImpact = other.CostImpact
	}
	if p.AllowedValues == nil {
		p.AllowedValues = other.AllowedValues
	}
	p.Required = p.Required || other.Required
}

func (p SortPoliciesByDisplayName) Len() int {
	return len(p)
}

func (p SortPoliciesByDisplayName) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p SortPoliciesByDisplayName) Less(i, j int) bool {
	return strings.Compare(p[i].DisplayName, p[j].DisplayName) < 0
}

func (p SortPolicyParametersByInternalName) Len() int {
	return len(p)
}

func (p SortPolicyParametersByInternalName) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p SortPolicyParametersByInternalName) Less(i, j int) bool {
	return strings.Compare(p[i].InternalName, p[j].InternalName) < 0
}
