package rest

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	HTTP_POST    = "POST"
	HTTP_GET     = "GET"
	HTTP_PUT     = "PUT"
	HTTP_PATCH   = "PATCH"
	HTTP_DELETE  = "DELETE"
	HTTP_HEAD    = "HEAD"
	HTTP_OPTIONS = "OPTIONS"
)

type ApiClient struct {
	restClient *resty.Client
}

func CreateApiClient() *ApiClient {
	client := resty.New()
	return &ApiClient{restClient: client}
}

func (c *ApiClient) WithBaseURL(baseUrl string) *ApiClient {
	c.restClient.SetBaseURL(baseUrl)
	return c
}

func (c *ApiClient) WithTimeout(timeout time.Duration) *ApiClient {
	c.restClient.SetTimeout(timeout)
	return c
}

func (c *ApiClient) WithRetryCount(retryCount int) *ApiClient {
	c.restClient.SetRetryCount(retryCount)
	return c
}

func (c *ApiClient) WithRetryInterval(retryInterval time.Duration) *ApiClient {
	c.restClient.SetRetryWaitTime(retryInterval)
	return c
}

func (c *ApiClient) WithDebug(debug bool) *ApiClient {
	c.restClient.SetDebug(debug)
	return c
}

func (c *ApiClient) WithTlsConfig(tlsConfig *tls.Config) *ApiClient {
	c.restClient.SetTLSClientConfig(tlsConfig)
	return c
}

func (client *ApiClient) DoLogin(authUrl string, oauthBody map[string]string, res any) (resp *resty.Response, err error) {
	request := client.restClient.R().
		SetFormData(oauthBody).
		SetResult(res)
	return request.Post(authUrl)
}

func (client *ApiClient) DoPost(accessToken, pathUrl string,
	data any, result any, pathParams map[string]string) (*resty.Response, error) {
	request := client.restClient.R().
		SetAuthToken(accessToken).
		SetPathParams(pathParams).
		SetBody(data).
		SetResult(result).
		SetHeader("Content-Type", "application/json")
	return doExecute(request, HTTP_POST, pathUrl)
}

func (client *ApiClient) DoGet(accessToken, pathUrl string,
	result any, pathParams map[string]string) (*resty.Response, error) {
	request := client.restClient.R().
		SetAuthToken(accessToken).
		SetPathParams(pathParams).
		SetResult(result).
		SetHeader("Content-Type", "application/json")
	return doExecute(request, HTTP_GET, pathUrl)
}

func (client *ApiClient) DoQuery(accessToken, pathUrl string,
	result any, pathParams map[string]string, queryParams map[string]string) (*resty.Response, error) {
	request := client.restClient.R().
		SetAuthToken(accessToken).
		SetPathParams(pathParams).
		SetQueryParams(queryParams).
		SetResult(result).
		SetHeader("Content-Type", "application/json")
	return doExecute(request, HTTP_GET, pathUrl)
}

func (client *ApiClient) DoPut(accessToken, pathUrl string,
	data any, result any, pathParams map[string]string) (*resty.Response, error) {
	request := client.restClient.R().
		SetAuthToken(accessToken).
		SetPathParams(pathParams).
		SetBody(data).
		SetResult(result).
		SetHeader("Content-Type", "application/json")
	return doExecute(request, HTTP_PUT, pathUrl)
}

func (client *ApiClient) DoDelete(accessToken, pathUrl string,
	result any, pathParams map[string]string) (*resty.Response, error) {
	request := client.restClient.R().
		SetAuthToken(accessToken).
		SetPathParams(pathParams).
		SetResult(result).
		SetHeader("Content-Type", "application/json")
	return doExecute(request, HTTP_DELETE, pathUrl)
}

func (client *ApiClient) DoExecute(
	method, urlPath, accessToken string,
	body any,
	result any,
	pathParams map[string]string,
	queryParams map[string]string,
	headers map[string]string) (*resty.Response, error) {

	request := client.restClient.R().
		SetAuthToken(accessToken).
		SetPathParams(pathParams).
		SetQueryParams(queryParams).
		SetBody(body).
		SetResult(result).
		SetHeaders(headers)
	if headers == nil {
		request.SetHeaders(map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		})
	}
	return doExecute(request, method, urlPath)
}

func doExecute(request *resty.Request, method, urlPath string) (*resty.Response, error) {
	switch method {
	case HTTP_POST, HTTP_GET, HTTP_PUT, HTTP_PATCH, HTTP_DELETE, HTTP_HEAD, HTTP_OPTIONS:
		return request.Execute(method, urlPath)
	default:
		return nil, fmt.Errorf("unsupported request: %s", method)
	}
}
