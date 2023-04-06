package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type HttpClient struct {
	httpClient *http.Client
	cancelFunc context.CancelFunc
	context    context.Context
}

type ReqConfig struct {
	Payload interface{}
	Cookies []*http.Cookie
	Method  string
	HttpUrl string
	Header  map[string]string
}

const defaultHttpClientTimeout = 30 * time.Second

// newClient configures and returns a new client
func newClient() (c *HttpClient) {
	// Initialize context use to cancel all pending requests when shutdown request is made.
	ctx, cancel := context.WithCancel(context.Background())

	return &HttpClient{
		context:    ctx,
		cancelFunc: cancel,
		httpClient: &http.Client{
			Timeout:   defaultHttpClientTimeout,
			Transport: http.DefaultTransport.(*http.Transport).Clone(),
		},
	}
}

func (c *HttpClient) getRequestBody(method string, body interface{}) ([]byte, error) {
	if body == nil {
		return nil, nil
	}

	if method == http.MethodPost {
		if requestBody, ok := body.([]byte); ok {
			return requestBody, nil
		}
	} else if method == http.MethodGet {
		if requestBody, ok := body.(map[string]string); ok {
			params := url.Values{}
			for key, val := range requestBody {
				params.Add(key, val)
			}
			return []byte(params.Encode()), nil
		}
	}

	return nil, errors.New("invalid request body")
}

// query prepares and process HTTP request to backend resources.
func (c *HttpClient) query(reqConfig *ReqConfig) (rawData []byte, resp *http.Response, err error) {
	// package the request body for POST and PUT requests
	var requestBody []byte
	if reqConfig.Payload != nil {
		requestBody, err = c.getRequestBody(reqConfig.Method, reqConfig.Payload)
		if err != nil {
			return nil, nil, err
		}
	}

	// package request URL for GET requests.
	if reqConfig.Method == http.MethodGet && requestBody != nil {
		reqConfig.HttpUrl += "?" + string(requestBody)
	}

	// Create http request
	req, err := http.NewRequestWithContext(c.context, reqConfig.Method, reqConfig.HttpUrl, bytes.NewReader(requestBody))
	if err != nil {
		return nil, nil, fmt.Errorf("error creating http request: %v", err)
	}

	if req == nil {
		return nil, nil, errors.New("error: nil request")
	}

	if reqConfig.Method == http.MethodPost || reqConfig.Method == http.MethodPut {
		req.Header.Add("Content-Type", "application/json;charset=utf-8")
	} else {
		req.Header.Add("Accept", "application/json")
	}

	for k, v := range reqConfig.Header {
		req.Header.Add(k, v)
	}

	for _, cookie := range reqConfig.Cookies {
		req.AddCookie(cookie)
	}

	// Send request
	resp, err = c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp, fmt.Errorf("error: status: %v", resp.Status)
	}

	return nil, resp, nil
}

// HttpRequest queries the API provided in the ReqConfig object and converts
// the returned json(Byte data) into an respObj interface.
func HttpRequest(reqConfig *ReqConfig, respObj interface{}) error {
	client := newClient()

	_, httpResp, err := client.query(reqConfig)
	if err != nil {
		return err
	}

	dec := json.NewDecoder(httpResp.Body)
	if err := dec.Decode(respObj); err != nil {
		return err
	}

	httpResp.Body.Close()
	return nil
}
