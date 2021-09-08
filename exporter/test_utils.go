package exporter

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func SkipTest() bool {
	v := os.Getenv("TEST")
	return strings.ToLower(v) != "true"
}

func PrettyPrint(v interface{}) error {
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", string(b))
	return nil
}

func TestResourceDir() string {
	return "../test_resources"
}
