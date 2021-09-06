package exporter

import (
	"fmt"
)

func MissingInputError(paramName string, target string) error {
	return fmt.Errorf("required input '%s' is missing for %s", paramName, target)
}

func UnexpectedValueError(name string, value interface{}) error {
	return fmt.Errorf("unexpected value '%+v' for '%s'", value, name)
}
