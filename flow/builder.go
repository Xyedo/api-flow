package flow

import (
	"bytes"
	"log"
	"net/url"
	"text/template"
)

func (p API) GenerateURL(savedResponseKeys map[string]any) *url.URL {
	t := template.New("route")
	route := parseTemplateOrDefault(
		t,
		savedResponseKeys,
		p.Route,
		p.Route,
	)

	url, err := url.Parse(route)
	if err != nil {
		log.Fatalf("failed parse url for this route %s\n", p.Route)
	}

	return url
}

func (p API) SetQueryParams(url *url.URL, savedResponseKeys map[string]any) {
	if p.QueryParams == nil {
		return
	}
	tmpl := template.New("queryParams")
	q := url.Query()
	for k, vs := range p.QueryParams {
		k = parseTemplateOrDefault(tmpl, savedResponseKeys, k, k)
		for _, v := range vs {
			v = parseTemplateOrDefault(tmpl, savedResponseKeys, v, v)
			q.Add(k, v)
		}
	}
	url.RawQuery = q.Encode()
}

func (p API) GenerateBody(savedResponseKeys map[string]any) string {
	tmpl := template.New("body")

	body, err := p.GetStringifyBody()
	if err != nil {
		log.Fatalln("invalid json body")
	}

	return parseTemplateOrDefault(tmpl, savedResponseKeys, body, body)
}

func parseTemplateOrDefault(tmpl *template.Template, data map[string]any, valueToParse, defaultValue string) string {
	t, err := tmpl.Parse(valueToParse)
	if err != nil {
		return defaultValue
	}

	var buff bytes.Buffer
	err = t.Execute(&buff, data)
	if err != nil {
		return defaultValue
	}

	return buff.String()
}
