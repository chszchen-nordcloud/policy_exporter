package exporter

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	BASE_DIR = "Governance/Policies/Templates"
)

type TemplateFileKind int

const (
	PolicyDefinition TemplateFileKind = iota
	PolicyParameters
	InitiativeDefinition
	InitiativeParameters
)

type policyTemplate struct {
	DisplayName    string                             `json:"displayName"`
	Description    string                             `json:"description"`
	AssignmentName string                             `json:"assignmentName"`
	Mode           string                             `json:"mode"`
	Parameters     map[string]policyTemplateParameter `json:"parameters"`
}

type policyTemplateParameter struct {
	Type     string                          `json:"type"`
	Metadata policyTemplateParameterMetadata `json:"metadata"`
}

type policyTemplateParameterMetadata struct {
	DisplayName   string        `json:"displayName"`
	Description   string        `json:"description"`
	StrongType    string        `json:"strongType"`
	AllowedValues []interface{} `json:"allowedValues"`
	DefaultValue  interface{}   `json:"defaultValue"`
}

type templateFileInfo struct {
	Kind              TemplateFileKind
	ParameterFileName string
}

func ReadCustomPoliciesFromLocalRepository(repositoryDir string) ([]Policy, error) {
	rootDir := filepath.Join(repositoryDir, BASE_DIR)
	policies := make([]Policy, 0, 64)
	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, previousErr error) error {
		if previousErr != nil {
			return previousErr
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".json" {
			return nil
		}

		templateInfo, err := newTemplateFileInfo(d.Name())
		if err != nil {
			fmt.Println(err)
			return nil
		}

		var parameterFilePath string
		switch templateInfo.Kind {
		case InitiativeParameters, PolicyParameters:
			return nil
		case PolicyDefinition, InitiativeDefinition:
			parameterFilePath = filepath.Join(filepath.Dir(path), templateInfo.ParameterFileName)
		}

		if _, err := os.Stat(parameterFilePath); os.IsNotExist(err) {
			return nil
		}

		b, err := ioutil.ReadFile(parameterFilePath)
		if err != nil {
			return err
		}
		policyParamsTmpl := policyTemplate{}
		err = json.Unmarshal(b, &policyParamsTmpl)
		if err != nil {
			return err
		}

		policy := policyParamsTmpl.ToPolicy()

		category, err := getCategoryFromFilePath(rootDir, path)
		if err != nil {
			return err
		}
		policy.Category = category

		err = fillMissingInformationFromPolicyDefinition(&policy, templateInfo.Kind, path)
		if err != nil {
			return err
		}

		policies = append(policies, policy)
		return nil
	})
	return policies, err
}

func fillMissingInformationFromPolicyDefinition(policy *Policy, kind TemplateFileKind, definitionPath string) error {
	if kind != PolicyDefinition {
		return nil
	}

	for _, param := range policy.Parameters {
		if param.InternalName == "effect" {
			return nil
		}
	}

	b, err := ioutil.ReadFile(definitionPath)
	if err != nil {
		return err
	}
	m := make(map[string]interface{})
	err = json.Unmarshal(b, &m)
	if err != nil {
		return err
	}
	obj := JSONObject(m)
	effect, err := obj.GetString("effect")
	if err != nil {
		return nil
	}

	policy.Parameters = append(policy.Parameters, PolicyParameter{
		Type:         "string",
		InternalName: "effect",
		DefaultValue: effect,
	})
	return nil
}

func getCategoryFromFilePath(rootDir string, path string) (string, error) {
	relpath := strings.TrimPrefix(path, rootDir)
	sep := string(filepath.Separator)
	if strings.HasPrefix(relpath, sep) {
		relpath = relpath[1:]
	}
	sepIdx := strings.Index(relpath, sep)
	if sepIdx == -1 {
		return "", fmt.Errorf("template file is supposed to be under a category folder: %s", path)
	}
	return relpath[0:sepIdx], nil
}

func newTemplateFileInfo(name string) (*templateFileInfo, error) {
	invalidFileName := func() error {
		return fmt.Errorf("invalid template file name: %s", name)
	}
	rightDotIdx := strings.LastIndex(name, ".")
	if rightDotIdx == -1 {
		return nil, invalidFileName()
	}
	secondRightDotIdx := strings.LastIndex(name[0:rightDotIdx], ".")
	if secondRightDotIdx == -1 {
		return nil, invalidFileName()
	}

	var kind TemplateFileKind
	switch strings.ToLower(name[secondRightDotIdx+1 : rightDotIdx]) {
	case "initiative":
		kind = InitiativeDefinition
	case "rules":
		kind = PolicyDefinition
	case "parameters":
		if strings.HasPrefix(name, ".") {
			kind = InitiativeParameters
		} else {
			kind = PolicyParameters
		}
	}

	var startIdx int
	switch kind {
	case InitiativeParameters, InitiativeDefinition:
		startIdx = 1
	case PolicyParameters, PolicyDefinition:
		startIdx = 0
	}
	dotIdx := strings.Index(name[startIdx:], ".")
	if dotIdx == -1 || dotIdx == startIdx {
		return nil, invalidFileName()
	}

	return &templateFileInfo{
		Kind:              kind,
		ParameterFileName: fmt.Sprintf("%s.parameters.json", name[0:secondRightDotIdx]),
	}, nil
}

func (p *policyTemplate) ToPolicy() Policy {
	params := make([]PolicyParameter, 0, len(p.Parameters))
	for name, paramDef := range p.Parameters {
		param := paramDef.ToPolicyParameter()
		if param.InternalName == "" {
			param.InternalName = name
		}
		params = append(params, param)
	}
	return Policy{
		DisplayName: p.DisplayName,
		Description: p.Description,
		Parameters:  params,
	}
}

func (p *policyTemplateParameter) ToPolicyParameter() PolicyParameter {
	return PolicyParameter{
		Type:          p.Type,
		InternalName:  p.Metadata.DisplayName,
		Description:   p.Metadata.Description,
		DefaultValue:  p.Metadata.DefaultValue,
		AllowedValues: p.Metadata.AllowedValues,
	}
}
