package exporter

type Category struct {
	Name     string   `json:"name"`
	Policies []Policy `json:"policies"`
}

type Policy struct {
	DisplayName      string                `json:"name" yaml:"DisplayName"`
	ResourceID       string                `json:"-" yaml:"ResourceID"`
	Justification    string                `json:"-" yaml:"Justification"`
	ManagementGroups map[string]Attachment `json:"managementGroups" yaml:"ManagementGroups"`
}

type Attachment struct {
	Enabled    bool              `json:"enabled" yaml:"-"`
	Parameters map[string]string `json:"parameters" yaml:"-"`
	Location   string            `json:"-" yaml:"-"`
	Effect     string            `json:"-" yaml:"Effect"`
}

type PolicySetParameters = map[string]PolicySetParameterValue

type PolicySetParameterValue struct {
	Value string `json:"value"`
}
