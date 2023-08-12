package flow

import (
	"encoding/json"
	"os"
)



func ReadConfigFile() (Flow, error) {
	data, err := os.ReadFile(".flow.json")
	if err != nil {
		return Flow{}, err
	}
	var flow Flow
	err = json.Unmarshal(data, &flow)
	if err != nil {
		return Flow{}, err
	}

	return flow, nil
}
