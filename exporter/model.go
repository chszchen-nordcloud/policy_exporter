package exporter

import (
	"context"
)

type PolicyDefinitionProvider struct {
	BuiltInPolicyReader         PolicyReader
	CustomPolicyReader          PolicyReader
	ASCPolicySetParameterReader PolicySetParameterReader
}

type PolicyDefinitionExporter struct {
	PolicyExporter             PolicyExporter
	BuiltInPolicyExporter      PolicyExporter
	CustomPolicyExporter       PolicyExporter
	PolicySetParameterExporter PolicySetParameterExporter
}

type PolicyReader = func(ctx context.Context) ([]Policy, error)

type PolicySetParameterReader = func(ctx context.Context) ([]PolicyParameter, error)

type PolicyExporter = func(policies []Policy, targetDir string) error

type PolicySetParameterExporter = func(policySetParameters []PolicyParameter, targetDir string) error

type Category struct {
	Name     string   `json:"name"`
	Policies []Policy `json:"policies"`
}

type Policy struct {
	DisplayName      string                `json:"name" yaml:"DisplayName"`
	ResourceID       string                `json:"-" yaml:"ResourceID"`
	Justification    string                `json:"-" yaml:"Justification"`
	Description      string                `json:"-" yaml:"-"`
	Parameters       []PolicyParameter     `json:"-" yaml:"-"`
	ManagementGroups map[string]Attachment `json:"managementGroups" yaml:"ManagementGroups"`
	Optional         bool                  `json:"-" yaml:"-"`
}

type Attachment struct {
	Enabled    bool              `json:"enabled" yaml:"-"`
	Parameters map[string]string `json:"parameters" yaml:"-"`
	Location   string            `json:"-" yaml:"-"`
	Effect     string            `json:"-" yaml:"Effect"`
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
	ManagementGroups map[string]string
}

type PolicyParameterValue struct {
	Value string `json:"value"`
}

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
	p.Optional = p.Optional && other.Optional
	managementGroups := &p.ManagementGroups
	for k, v := range other.ManagementGroups {
		if existing, ok := (*managementGroups)[k]; ok {
			existing.Merge(v)
			(*managementGroups)[k] = existing
		} else {
			(*managementGroups)[k] = existing
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
}
