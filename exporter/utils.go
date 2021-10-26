package exporter

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
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

func (o *JSONObject) MustGetString(keys ...string) string {
	v, err := o.GetString(keys...)
	if err != nil {
		panic(err)
	}
	return v
}

func (o *JSONObject) GetArrayElementAsObject(match func(*JSONObject) bool, keys ...string) (*JSONObject, error) {
	arr, err := o.GetArray(keys...)
	if err != nil {
		return nil, err
	}
	for i := range arr {
		obj := JSONObject(arr[i].(map[string]interface{}))
		if match(&obj) {
			return &obj, nil
		}
	}
	return nil, fmt.Errorf("no element matches in array at %v", keys)
}

func ParseArrayValue(s string) (interface{}, error) {
	s = strings.TrimSpace(s)
	end := len(s)
	if end < 2 {
		return nil, fmt.Errorf("not valid array value as there are less than 2 characters: %s", s)
	}

	if end == 2 {
		return []interface{}{}, nil
	}

	s = strings.TrimSpace(s[1 : len(s)-1])
	if len(s) == 0 {
		return []interface{}{}, nil
	}

	result := make([]interface{}, 0, 8)
	var token string
	var valueType string
	tokens := NewTokenizer(s)
	if tokens.HasNext() {
		token = tokens.Next()
		vt, err := determineValueType(token)
		if err != nil {
			return nil, err
		}
		valueType = vt
	} else {
		return nil, fmt.Errorf("expected at least one token: %s", s)
	}
	converted, err := parseSingleParameterValue(valueType, token)
	if err != nil {
		return nil, err
	}
	result = append(result, converted)
	for tokens.HasNext() {
		token := tokens.Next()
		converted, err := parseSingleParameterValue(valueType, token)
		if err != nil {
			return nil, err
		}
		result = append(result, converted)
	}
	return result, nil
}

type Tokenizer struct {
	s               string
	pos             int
	end             int
	valueQuoteMarks []byte
	delimiter       byte
}

func NewTokenizer(s string) Tokenizer {
	return Tokenizer{
		s:               s,
		pos:             0,
		end:             len(s),
		valueQuoteMarks: []byte{'"', '\''},
		delimiter:       ',',
	}
}

func (t *Tokenizer) HasNext() bool {
	return t.pos < t.end
}

func (t *Tokenizer) Next() string {
	s := t.s
	start := t.pos
	end := t.end
	var quoteMark byte
	unsetQuoteMark := func() { quoteMark = '0' }
	isQuoteMarkSet := func() bool { return quoteMark == '0' }
	unsetQuoteMark()
	for i := start; i < t.end; i++ {
		c := s[i]
		if t.isQuoteMark(c) {
			if quoteMark == c {
				unsetQuoteMark()
				end = i
			} else {
				quoteMark = c
				start = i + 1
			}
		}
		if isQuoteMarkSet() && t.isDelimiter(c) {
			if end > i {
				end = i
			}
			return t.consume(start, end, i+1)
		}
	}
	return t.consume(start, end, t.end)
}

func (t *Tokenizer) consume(from int, to int, pos int) string {
	result := t.s[from:to]
	t.pos = pos
	return result
}

func (t *Tokenizer) isQuoteMark(c byte) bool {
	for _, q := range t.valueQuoteMarks {
		if c == q {
			return true
		}
	}
	return false
}

func (t *Tokenizer) isDelimiter(c byte) bool {
	return c == t.delimiter
}

const (
	ParameterTypeInteger = "integer"
	ParameterTypeString  = "string"
	ParameterTypeBool    = "boolean"
)

func determineValueType(v string) (string, error) {
	s := strings.TrimSpace(v)
	if len(s) == 0 {
		return "", fmt.Errorf("empty array element value: %s", s)
	}

	end := len(s)
	for end > 1 && s[0] == s[end-1] && (s[0] == '"' || s[0] == '\'') {
		s = s[1 : end-1]
		end = len(s)
	}

	if end == 0 {
		return "", fmt.Errorf("empty array element value: %s", s)
	}

	_, err := strconv.Atoi(s)
	if err == nil {
		return ParameterTypeInteger, nil
	}
	if strings.EqualFold(s, "true") || strings.EqualFold(s, "false") {
		return ParameterTypeBool, nil
	}
	return ParameterTypeString, nil
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
