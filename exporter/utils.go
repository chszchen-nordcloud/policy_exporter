package exporter

import (
	"encoding/json"
	"fmt"
	"os"
)

type JSONObject map[string]interface{}

func NewJSONObjectFromFile(file string) (JSONObject, error) {
	result := make(JSONObject)
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (o *JSONObject) GetArray(keys ...string) ([]interface{}, error) {
	v, err := mapLookup(map[string]interface{}(*o), keys...)
	if err != nil {
		return nil, err
	}
	m, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("value %v is not an array", v)
	}
	return m, nil
}

func (o *JSONObject) GetObject(keys ...string) (JSONObject, error) {
	v, err := mapLookup(map[string]interface{}(*o), keys...)
	if err != nil {
		return nil, err
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("value %v is not a map", v)
	}
	return m, nil
}

func (o *JSONObject) GetString(keys ...string) (string, error) {
	v, err := mapLookup(map[string]interface{}(*o), keys...)
	if err != nil {
		return "", err
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("value %v is not a string", v)
	}
	return s, nil
}

func mapLookup(obj interface{}, keys ...string) (interface{}, error) {
	if len(keys) == 0 {
		return obj, nil
	}
	m, ok := obj.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("keys %v is invalid against object %v", keys, obj)
	}
	return mapLookup(m[keys[0]], keys[1:]...)
}

func ConvertTo(from interface{}, to interface{}) error {
	b, err := json.Marshal(from)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, to)
}

func ToJSONPretty(v interface{}) (string, error) {
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
