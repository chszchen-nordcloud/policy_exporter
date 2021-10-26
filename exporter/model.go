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

type UniqueResource interface {
	ID() interface{}

	Merge(other UniqueResource)
}

// Category is used in Azure LandingZone JSON parameters files.
type Category struct {
	Name     string   `json:"name"`
	Policies []Policy `json:"policies"`
}

type Policy struct {
	Category         string                `json:"-" yaml:"Category"`
	DisplayName      string                `json:"name" yaml:"DisplayName"`
	Effect           string                `json:"-" yaml:"-"`
	Recommend        bool                  `json:"-"`
	IsInitiative     bool                  `json:"-" yaml:"-"`
	InitiativeIDs    []string              `json:"-" yaml:"-"`
	ResourceID       string                `json:"definitionId,omitempty" yaml:"ResourceID"`
	Justification    string                `json:"-" yaml:"Justification"`
	CostImpact       string                `json:"-" yaml:"CostImpact"`
	Description      string                `json:"info,omitempty" yaml:"Description"`
	Parameters       []PolicyParameter     `json:"-" yaml:"-"`
	ManagementGroups map[string]Attachment `json:"managementGroups" yaml:"ManagementGroups"`
}

// Attachment attaches a policy to a management group.
type Attachment struct {
	Enabled      bool                   `json:"enabled" yaml:"Enabled"`
	Parameters   map[string]interface{} `json:"parameters,omitempty" yaml:"Parameters"`
	Location     string                 `json:"location" yaml:"Location"`
	Effect       string                 `json:"-" yaml:"Effect"`
	LinkedScopes []string               `json:"linkedScopes,omitempty"`
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
}

// PolicyParameterValue is used in Azure LandingZone JSON parameter file.
type PolicyParameterValue struct {
	Value interface{} `json:"value"`
}

func (p *Policy) ID() interface{} {
	return p.DisplayName
}

func (p *PolicyParameter) ID() interface{} {
	return p.InternalName
}

// Merge merges two policy definitions into one.
func (p *Policy) Merge(otherResource UniqueResource) {
	other := otherResource.(*Policy)
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
	if p.ManagementGroups == nil {
		p.ManagementGroups = make(map[string]Attachment)
	}
	p.Recommend = p.Recommend && other.Recommend
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

func (p *Policy) GetParametersForExport() []PolicyParameter {
	result := make([]PolicyParameter, 0, len(p.Parameters))
	for _, param := range p.Parameters {
		if !strings.HasPrefix(param.InternalName, "*") {
			result = append(result, param)
		}
	}
	return result
}

func (p *Policy) GetParameter(internalName string) *PolicyParameter {
	var result *PolicyParameter
	for i := range p.Parameters {
		if p.Parameters[i].InternalName == internalName {
			result = &p.Parameters[i]
			break
		}
	}
	return result
}

func (p *Policy) NormalizeEffectParameter() {
	if p.IsInitiative {
		return
	}
	effectParam := p.GetParameter("effect")
	if effectParam != nil {
		if effectParam.DefaultValue == nil {
			effectParam.DefaultValue = p.Effect
		}
	} else {
		p.Parameters = append(p.Parameters, PolicyParameter{
			InternalName: "*effect",
			Type:         "string",
			DefaultValue: p.Effect,
		})
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
	if a.LinkedScopes == nil {
		a.LinkedScopes = other.LinkedScopes
	}
	for k, v := range other.Parameters {
		a.Parameters[k] = v
	}
}

func (p *PolicyParameter) Merge(otherResource UniqueResource) {
	other := otherResource.(*PolicyParameter)
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
	if p.ManagementGroups == nil {
		p.ManagementGroups = make(map[string]interface{})
	}
	for k, v := range other.ManagementGroups {
		if _, ok := p.ManagementGroups[k]; !ok {
			p.ManagementGroups[k] = v
		}
	}
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

func Unique(valuesWithID []UniqueResource) []UniqueResource {
	m := make(map[interface{}]UniqueResource)
	for i := range valuesWithID {
		k := valuesWithID[i].ID()
		v, ok := m[k]
		if ok {
			v.Merge(valuesWithID[i])
		} else {
			m[k] = valuesWithID[i]
		}
	}

	uniqueValues := make([]UniqueResource, 0, len(m))
	for _, v := range m {
		uniqueValues = append(uniqueValues, v)
	}
	return uniqueValues
}
