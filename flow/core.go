package flow

import "encoding/json"

type Flow struct {
	PreCondition  Precondition  `json:"precondition"`
	Integration   Integration   `json:"integration"`
	PostCondition PostCondition `json:"postcondition"`
}
type Precondition struct {
	BaseURL     string             `json:"baseUrl"`
	BearerToken string             `json:"bearerToken"`
	Steps       []PreConditionStep `json:"steps"`
}
type PreConditionStep struct {
	API
	ResponseSave
}
type API struct {
	Method      string              `json:"method" jsonschema:"enum=get,enum=post,enum=put,enum=patch,enum=delete"`
	Route       string              `json:"route"`
	QueryParams map[string][]string `json:"queryParams"`
	Body        map[string]any      `json:"body"`
	StatusCode  int                 `json:"statusCode"`
}

func (a API) GetStringifyBody() (string, error) {
	b, err := json.Marshal(a.Body)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

type ResponseSave struct {
	Keys map[string]string `json:"saveKeys"`
}

type Integration struct {
	BearerToken string `json:"bearerToken"`
	API
	Test
}
type PostCondition struct {
	BearerToken string `json:"bearerToken"`
	BaseURL     string `json:"baseUrl"`
	Steps       []API  `json:"steps"`
}

type Test struct {
	Prefix         string         `json:"prefix"`
	Level          string         `json:"level" jsonschema:"enum=warn,enum=error"`
	MatchKeyValue  map[string]any `json:"matchKeyValue"`
	MatchKeyExists []string       `json:"matchKeyExists"`
}
