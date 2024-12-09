package cmd

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

func HttpClientNewTransPort() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   2 * time.Second,
			KeepAlive: 60 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:           100,
		MaxIdleConnsPerHost:    100,
		IdleConnTimeout:        10 * time.Second,
		TLSHandshakeTimeout:    6 * time.Second,
		MaxResponseHeaderBytes: 8192,
		ResponseHeaderTimeout:  time.Millisecond * 5000,
		DisableKeepAlives:      false,
	}
}

// Don't forget add https:// or http
func HttpGet(url string) (*http.Response, string, error) {
	response, err := Global.HttpClient.Get(url)
	if err != nil {
		//Logger.Error(err)
		return response, "", err
	}
	defer response.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		Logger.Error("Error reading response body:", err)
		return response, "", err
	}
	return response, string(body), err
}

func PrettyPrintJson(jsonStr string) (string, error) {
	var data interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return "", err
	}

	// Marshal the data with indentation
	prettyJSON, err := json.MarshalIndent(data, "", string(byte('\t')))
	if err != nil {
		return "", err
	}
	return string(prettyJSON), err
}
