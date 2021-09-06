package exporter

import (
	"context"
)

type PolicyDefinitionProvider struct {
	PolicyReader             PolicyReader
	PolicySetParameterReader PolicySetParameterReader
}

type PolicyDefinitionExporter struct {
	PolicyExporter             PolicyExporter
	PolicySetParameterExporter PolicySetParameterExporter
}

type PolicyReader = func(ctx context.Context) ([]Policy, error)

type PolicySetParameterReader = func(ctx context.Context) ([]PolicySetParameter, error)

type PolicyExporter = func(ctx context.Context, policies []Policy, filepath string) error

type PolicySetParameterExporter = func(ctx context.Context, policySetParameters []PolicySetParameter, filepath string) error

type Category struct {
	Name     string   `json:"name"`
	Policies []Policy `json:"policies"`
}

type Policy struct {
	DisplayName      string                `json:"name" yaml:"DisplayName"`
	ResourceID       string                `json:"-" yaml:"ResourceID"`
	Justification    string                `json:"-" yaml:"Justification"`
	Description      string                `json:"-" yaml:"-"`
	ManagementGroups map[string]Attachment `json:"managementGroups" yaml:"ManagementGroups"`
}

type Attachment struct {
	Enabled    bool              `json:"enabled" yaml:"-"`
	Parameters map[string]string `json:"parameters" yaml:"-"`
	Location   string            `json:"-" yaml:"-"`
	Effect     string            `json:"-" yaml:"Effect"`
}

type PolicySetParameter struct {
	InternalName  string
	DisplayName   string
	Description   string
	DefaultValue  interface{}
	Justification string
	CostImpact    string
	AllowedValues []interface{}
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

func (p *PolicySetParameter) Merge(other PolicySetParameter) {
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
