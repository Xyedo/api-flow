package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"integration-workflow/flow"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
)

func main() {
	f, err := flow.ReadConfigFile()
	if err != nil {
		log.Fatalln(err)
	}

	client := retryablehttp.NewClient()
	savedResponseKeys := make(map[string]any)
	// Precondition
	{
		for i, precondition := range f.PreCondition.Steps {
			url := precondition.GenerateURL(savedResponseKeys)
			precondition.SetQueryParams(url, savedResponseKeys)

			method, err := parseApiMethod(precondition.Method)
			if err != nil {
				log.Fatalln(fmt.Errorf("%v: on %s\nsupported method is POST,GET,PATCH,PUT,DELETE", err, precondition.Method))
			}

			var req *http.Request
			if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
				body := precondition.GenerateBody(savedResponseKeys)
				req, err = http.NewRequest(method, f.PreCondition.BaseURL+url.String(), strings.NewReader(body))
				if err != nil {
					log.Fatalf("invalid request on precondition index %d", i)
				}

				req.Header.Add("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(method, f.PreCondition.BaseURL+url.String(), nil)
				if err != nil {
					log.Fatalf("invalid request on precondition index %d", i)
				}
			}

			req.Header.Add("Authorization", "Bearer "+f.PreCondition.BearerToken)
			res, err := client.Do(&retryablehttp.Request{
				Request: req,
			})
			if err != nil {
				log.Fatalf("invalid operation on index %d", i)
			}
			respMap := make(map[string]any)
			err = json.NewDecoder(res.Body).Decode(&respMap)
			if err != nil && len(precondition.ResponseSave.Keys) != 0 {
				log.Fatalln(err)
			}

			if res.StatusCode != precondition.StatusCode {
				log.Fatalf("invalid expected statusCode on precondition index %d: expected %d but actual %d with body %v", i, precondition.StatusCode, res.StatusCode, respMap)
			}

			_ = res.Body.Close()

			for key, resp := range precondition.ResponseSave.Keys {
				paths := strings.Split(resp, "::")

				for _, path := range paths[:len(paths)-1] {
					if index, err := strconv.Atoi(path); err == nil {
						respArrInner, ok := respMap[path].([]map[string]any)
						if !ok {
							log.Fatalf("invalid preconditions responseSave keys on index key %s", key)
						}

						respMap = respArrInner[index]
					} else {
						respMapInner, ok := respMap[path].(map[string]any)
						if !ok {
							log.Fatalf("invalid preconditions responseSave keys on key %s", key)
						}

						respMap = respMapInner
					}

				}
				savedResponseKeys[key] = respMap[paths[len(paths)-1]]
			}

		}
	}

	// Integration
	if f.Integration.Route != "" {
		err := integration(f, savedResponseKeys, client)
		if err != nil {
			if f.Integration.Level == "error" {
				log.Fatalln(err)
			}

			if f.Integration.Level == "warn" {
				log.Println(err)
			}
		}
	}

	// Post Integration / Cleaning
	{
		for i, postCondition := range f.PostCondition.Steps {
			url := postCondition.GenerateURL(savedResponseKeys)
			postCondition.SetQueryParams(url, savedResponseKeys)

			method, err := parseApiMethod(postCondition.Method)
			if err != nil {
				log.Fatalln(fmt.Errorf("%v: on %s\nsupported method is POST,GET,PATCH,PUT,DELETE", err, postCondition.Method))
			}

			var req *http.Request
			if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
				body := postCondition.GenerateBody(savedResponseKeys)
				if err != nil {
					log.Fatalf("invalid body on %s", postCondition.Body)
				}

				req, err = http.NewRequest(method, f.PreCondition.BaseURL+url.String(), strings.NewReader(body))
				if err != nil {
					log.Fatalf("invalid request on precondition index %d", i)
				}

				req.Header.Add("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(method, f.PreCondition.BaseURL+url.String(), nil)
				if err != nil {
					log.Fatalf("invalid request on precondition index %d", i)
				}

			}

			req.Header.Add("Authorization", "Bearer "+f.PostCondition.BearerToken)

			res, err := client.Do(&retryablehttp.Request{
				Request: req,
			})
			if err != nil {
				log.Fatalf("invalid operation on index %d", i)
			}

			respMap := make(map[string]any)
			err = json.NewDecoder(res.Body).Decode(&respMap)
			if err != nil && res.StatusCode != postCondition.StatusCode {
				log.Fatalln(err)
			}

			if res.StatusCode != postCondition.StatusCode {
				log.Fatalf("invalid expected statusCode on postCondition: expected %d but actual %d with body %v", postCondition.StatusCode, res.StatusCode, respMap)

			}

			_ = res.Body.Close()

		}
	}

}

func integration(f flow.Flow, savedResponseKeys map[string]any, client *retryablehttp.Client) error {
	url := f.Integration.GenerateURL(savedResponseKeys)
	f.Integration.SetQueryParams(url, savedResponseKeys)

	method, err := parseApiMethod(f.Integration.Method)
	if err != nil {
		return fmt.Errorf("%v: on %s\nsupported method is POST,GET,PATCH,PUT,DELETE", err, f.Integration.Method)

	}

	var req *http.Request
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		body := f.Integration.GenerateBody(savedResponseKeys)

		req, err = http.NewRequest(method, f.Integration.Prefix+url.String(), strings.NewReader(body))
		if err != nil {
			return fmt.Errorf("invalid request on integration with err: %v", err)
		}

		req.Header.Add("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, f.Integration.Prefix+url.String(), nil)
		if err != nil {
			return fmt.Errorf("invalid request on integration with err: %v", err)
		}
	}

	req.Header.Add("Authorization", "Bearer "+f.Integration.BearerToken)
	res, err := client.Do(&retryablehttp.Request{
		Request: req,
	})
	if err != nil {
		return fmt.Errorf("invalid request on integration with err: %v", err)
	}
	respMap := make(map[string]any)
	err = json.NewDecoder(res.Body).Decode(&respMap)
	if err != nil {
		return err
	}

	if res.StatusCode != f.Integration.StatusCode {
		return fmt.Errorf("invalid expected statusCode on integration: expected %d but actual %d with body %v", f.Integration.StatusCode, res.StatusCode, respMap)

	}

	res.Body.Close()

	for _, key := range f.Integration.MatchKeyExists {
		if f.Integration.Prefix != "" {
			key = f.Integration.Prefix + "::" + key
		}
		paths := strings.Split(key, "::")

		for i := range paths {
			path := paths[i]

			if i == len(path)-1 {
				if _, ok := respMap[paths[i]]; !ok {
					if f.Integration.Level == "warn" {
						log.Printf("key %s not exist in the respose", key)
					}
					if f.Integration.Level == "error" {
						return fmt.Errorf("key %s not exist in the respose", key)
					}
				}
			}

			if index, err := strconv.Atoi(path); err == nil {
				respArrInner, ok := respMap[path].([]map[string]any)
				if !ok {
					return fmt.Errorf("invalid integration responseSave keys on index key %s", key)
				}

				respMap = respArrInner[index]
			} else {
				respMapInner, ok := respMap[path].(map[string]any)
				if !ok {
					return fmt.Errorf("invalid integration responseSave keys on key %s", key)
				}

				respMap = respMapInner
			}

		}
	}
	for key, value := range f.Integration.MatchKeyValue {
		if f.Integration.Prefix != "" {
			key = f.Integration.Prefix + "::" + key
		}

		paths := strings.Split(key, "::")

		for i := range paths {
			path := paths[i]

			if i == len(path)-1 {
				if respMap[paths[i]] != value {
					if f.Integration.Level == "warn" {
						log.Printf("not equal on key %s with expected value %v but have %v", key, value, respMap[paths[i]])
					}
					if f.Integration.Level == "error" {
						log.Fatalf("not equal on key %s with expected value %v but have %v", key, value, respMap[paths[i]])
					}
				}
			}

			if index, err := strconv.Atoi(path); err == nil {
				respArrInner, ok := respMap[path].([]map[string]any)
				if !ok {
					log.Fatalf("invalid integration responseSave keys on index key %s", key)
				}

				respMap = respArrInner[index]
			} else {
				respMapInner, ok := respMap[path].(map[string]any)
				if !ok {
					log.Fatalf("invalid integration responseSave keys on key %s", key)
				}

				respMap = respMapInner
			}

		}

	}

	return nil
}

func parseApiMethod(method string) (string, error) {
	if strings.EqualFold(method, http.MethodPost) {
		return http.MethodPost, nil
	}

	if strings.EqualFold(method, http.MethodGet) {
		return http.MethodGet, nil
	}

	if strings.EqualFold(method, http.MethodPatch) {
		return http.MethodPatch, nil
	}
	if strings.EqualFold(method, http.MethodPut) {
		return http.MethodPut, nil
	}

	if strings.EqualFold(method, http.MethodDelete) {
		return http.MethodDelete, nil
	}
	return "", errors.New("invalid method")
}
