package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
)

var (
	InvalidJSON = `{"invalid": json}`
)

// create JSON request body
func createJSONRequest(data interface{}) *bytes.Buffer {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return bytes.NewBuffer([]byte(""))
	}
	return bytes.NewBuffer(jsonData)
}

// create HTTP request with JSON body
func createJSONHTTPRequest(method, url string, data interface{}) *http.Request {
	req, err := http.NewRequest(method, url, createJSONRequest(data))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}
