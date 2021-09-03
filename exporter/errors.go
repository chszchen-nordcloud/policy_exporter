package exporter

import (
	"errors"
	"fmt"
)

func MissingInputError(paramName string, target string) error {
	return errors.New(fmt.Sprintf("required input '%s' is missing for %s", paramName, target))
}

func UnexpectedValueError(name string, value interface{}) error {
	return errors.New(fmt.Sprintf("unexpected value '%+v' for '%s'", value, name))
}