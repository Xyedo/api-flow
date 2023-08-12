package main

import (
	"encoding/json"
	"integration-workflow/flow"
	"os"

	"github.com/invopop/jsonschema"
)

func main() {
	s := jsonschema.Reflect(&flow.Flow{})
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		panic(err.Error())
	}
	err = os.WriteFile("flow.schema.json", data, 0644)
	if err != nil {
		panic(err.Error())
	}
}
