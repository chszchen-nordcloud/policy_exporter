package exporter

import (
	"context"
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
	Optional         bool                  `json:"-" yaml:"-"`
}

type Attachment map[string]interface{}

// StrictAttachment attaches a policy to a management group.
type StrictAttachment struct {
	Enabled    bool              `json:"enabled" yaml:"Enabled"`
	Parameters map[string]string `json:"parameters" yaml:"Parameters"`
	Location   string            `json:"location" yaml:"Location"`
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
	Optional         bool `json:"-" yaml:"-"`
}

// PolicyParameterValue is used in Azure LandingZone JSON parameter file.
type PolicyParameterValue struct {
	Value string `json:"value"`
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
	if p.Justification == "" {
		p.Justification = other.Justification
	}
	if p.CostImpact == "" {
		p.CostImpact = other.CostImpact
	}
	p.Optional = p.Optional && other.Optional
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

func (a *StrictAttachment) Merge(other StrictAttachment) {
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

func (a *Attachment) Merge(other Attachment) {
	for k, v := range other {
		(*a)[k] = v
	}
}
