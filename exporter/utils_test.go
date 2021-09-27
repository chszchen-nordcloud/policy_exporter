package exporter

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestToJsonPretty(t *testing.T) {
	policy := map[string]interface{}{
		"a": 1,
		"b": "b",
	}
	s, err := ToJSONPretty(policy)
	assert.NoError(t, err)
	assert.Equal(t, `{
	"a": 1,
	"b": "b"
}`, s)
	println(s)
}

func TestConvertTo(t *testing.T) {
	type Foo struct {
		Name string `json:"name"`
	}

	m := map[string]interface{}{
		"name": "foo",
	}
	foo := Foo{}
	err := ConvertTo(m, &foo)
	assert.NoError(t, err)

	m1 := make(map[string]interface{})
	err = ConvertTo(foo, &m1)
	assert.NoError(t, err)
	assert.Equal(t, m, m1)
}

func TestJsonObject(t *testing.T) {
	var m JSONObject
	err := json.Unmarshal([]byte(`{
                           "parameters": {
								"vmName": {
									"type": "string"
								},
								"vmResourceGroupName": {
									"type": "string"
								},
								"vmResourceGroupLocation": {
									"type": "string"
								},
								"vaultScheduledRuntime": {
									"type": "string"
								},
								"vaultDefaultRetentionDailyCount": {
									"type": "string"
								},
								"vaultDefaultRetentionWeeklyCount": {
									"type": "string"
								},
								"enableCRR": {
									"type": "bool",
									"defaultValue": true
								},
								"vaultStorageType": {
									"type": "string",
									"defaultValue": "GeoRedundant",
									"allowedValues": [
										"LocallyRedundant",
										"GeoRedundant"
									]
								}
							}
                        }`), &m)
	assert.NoError(t, err)

	s, err := m.GetString("parameters", "vaultStorageType", "type")
	assert.NoError(t, err)
	assert.Equal(t, "string", s)

	o, err := m.GetObject("parameters", "vaultDefaultRetentionWeeklyCount")
	assert.NoError(t, err)
	assert.Equal(t, JSONObject{"type": "string"}, o)
}
