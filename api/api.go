package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

var Transport = http.DefaultTransport.(*http.Transport)

func init() {
	Transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
}

type API struct {
	BaseUrl string
	Token   string
	URI     string
	Timeout time.Duration
	Header  http.Header

	rawQuery string
}

func New(baseUrl, token, uri string) *API {
	return &API{
		BaseUrl: baseUrl,
		Token:   token,
		URI:     uri,
		Timeout: 5 * time.Second,
		Header:  make(http.Header, 0),
	}
}

func NewWithTimeout(baseUrl, token, uri string, n time.Duration) *API {
	return &API{
		BaseUrl: baseUrl,
		Token:   token,
		URI:     uri,
		Timeout: n,
		Header:  make(http.Header, 0),
	}
}

func (api *API) Get(query url.Values) ([]byte, error) {
	api.rawQuery = query.Encode()
	return api.do("GET", nil)
}

func (api *API) Put(payload interface{}) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return data, err
	}

	return api.do("PUT", data)
}

func (api *API) Post(payload interface{}) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return data, err
	}

	return api.do("POST", data)
}

func (api *API) Delete(payload interface{}) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return data, err
	}

	return api.do("DELETE", data)
}

func (api *API) Do(method string, rawData []byte) ([]byte, error) {
	return api.do(method, rawData)
}

func (api *API) do(method string, rawData []byte) ([]byte, error) {
	body := bytes.NewBuffer(rawData)
	req, err := http.NewRequest(method, api.BaseUrl, body)
	if err != nil {
		return []byte(""), err
	}

	req.URL.Path = api.URI
	if api.rawQuery != "" {
		req.URL.RawQuery = api.rawQuery
	}
	req.Header.Set("API-TOKEN", api.Token)
	req.Header.Set("Content-Type", "application/json")
	if api.Header != nil {
		for k, v := range api.Header {
			req.Header[k] = v
		}
	}
	client := &http.Client{Transport: Transport, Timeout: api.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return []byte(""), err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
