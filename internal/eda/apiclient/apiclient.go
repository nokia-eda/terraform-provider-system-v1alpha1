package apiclient

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/nokia/eda/apps/terraform-provider-system/internal/eda/rest"
	"github.com/nokia/eda/apps/terraform-provider-system/internal/eda/utils"
)

const (
	// Constants
	KEY_CLIENT_ID      = "client_id"
	KEY_CLIENT_SECRET  = "client_secret"
	KEY_USERNAME       = "username"
	KEY_PASSWORD       = "password"
	KEY_GRANT_TYPE     = "grant_type"
	KEY_PASSWORD_GRANT = "password"
	KEY_REFRESH_GRANT  = "refresh_token"

	// URLs
	KEYCLOAK_URL = "/core/httpproxy/v1/keycloak"
	OAUTH_URL    = KEYCLOAK_URL + "/realms/%s/protocol/openid-connect/token"
	CLIENT_URL   = KEYCLOAK_URL + "/admin/realms/{realm}/clients"
)

type grant struct {
	AccessToken   string  `json:"access_token"`
	RefreshToken  string  `json:"refresh_token"`
	Scope         string  `json:"scope"`
	TokenType     string  `json:"token_type"`
	ExpiresInSecs float64 `json:"expires_in"`
	timestamp     *time.Time
}

type clientCredentials struct {
	authUrl      string
	clientId     string
	clientSecret string
	username     string
	password     string
}

type EdaApiClient struct {
	tokenLock     sync.Mutex
	cfg           *Config
	restClient    *rest.ApiClient
	edaCred       *clientCredentials
	keyCloakGrant *grant
	edaGrant      *grant
	logger        *slog.Logger
}

type Config struct {
	BaseURL           string        `json:"baseURL"`
	KcUsername        string        `json:"kcUsername"`
	KcPassword        string        `json:"kcPassword"`
	KcRealm           string        `json:"kcRealm"`
	KcClientID        string        `json:"kcClientId"`
	EdaUsername       string        `json:"edaUsername"`
	EdaPassword       string        `json:"edaPassword"`
	EdaRealm          string        `json:"edaRealm"`
	EdaClientID       string        `json:"edaClientId"`
	EdaClientSecret   string        `json:"edaClientSecret"`
	TlsSkipVerify     bool          `json:"tlsSkipVerify"`
	RestDebug         bool          `json:"restDebug"`
	RestTimeout       time.Duration `json:"restTimeout"`
	RestRetries       int           `json:"restRetries"`
	RestRetryInterval time.Duration `json:"restRetryInterval"`
}

func NewEdaApiClient(cfg *Config) (*EdaApiClient, error) {
	// Create a default logger with LOG_LEVEL set to info
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: utils.GetLogLevel().Level(),
	}))
	return NewEdaApiClientWithLogger(cfg, logger)
}

func NewEdaApiClientWithLogger(cfg *Config, logger *slog.Logger) (*EdaApiClient, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}

	client := &EdaApiClient{
		cfg: cfg,
		edaCred: &clientCredentials{
			authUrl:  fmt.Sprintf(OAUTH_URL, cfg.EdaRealm),
			clientId: cfg.EdaClientID,
			username: cfg.EdaUsername,
			password: cfg.EdaPassword,
		},
		keyCloakGrant: &grant{},
		edaGrant:      &grant{},
		logger:        logger,
	}
	client.restClient = rest.CreateApiClient().
		WithBaseURL(cfg.BaseURL).
		WithTimeout(cfg.RestTimeout).
		WithRetryCount(cfg.RestRetries).
		WithRetryInterval(cfg.RestRetryInterval).
		WithTlsConfig(&tls.Config{InsecureSkipVerify: cfg.TlsSkipVerify}).
		WithDebug(cfg.RestDebug)

	if cfg.EdaClientSecret != "" {
		client.edaCred.clientSecret = cfg.EdaClientSecret
		return client, nil
	}
	var err error
	client.edaCred.clientSecret, err = client.getClientSecret(cfg.EdaClientID)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (c *EdaApiClient) getEdaAccessToken() (string, error) {
	return c.getAccessToken(c.edaCred, c.edaGrant)
}

func (c *EdaApiClient) getAccessToken(cred *clientCredentials, grnt *grant) (string, error) {
	c.tokenLock.Lock()
	defer c.tokenLock.Unlock()

	expired := false
	var resp *resty.Response
	var err error
	if grnt.timestamp != nil && grnt.ExpiresInSecs != 0 {
		elapsed := time.Since(*grnt.timestamp).Seconds()
		c.logger.Debug("getAccessToken()", "authUrl", cred.authUrl, "timeElapsed", elapsed)
		expired = elapsed > grnt.ExpiresInSecs
	}
	c.logger.Debug("getAccessToken()", "authUrl", cred.authUrl,
		"grantExpired", expired, "accessToken", grnt.AccessToken, "refreshToken", grnt.RefreshToken)

	if !expired && grnt.AccessToken != "" {
		c.logger.Debug("getAccessToken()", "authUrl", cred.authUrl, "existingToken", grnt.AccessToken)
		return grnt.AccessToken, nil
	}
	if expired && grnt.RefreshToken != "" {
		resp, err = c.restClient.DoLogin(cred.authUrl, c.getOauthBody(cred, grnt.RefreshToken), grnt)
	} else {
		resp, err = c.restClient.DoLogin(cred.authUrl, c.getOauthBody(cred, ""), grnt)
	}
	if err != nil {
		return "", err
	}
	c.logger.Debug("getAccessToken()", "authUrl", cred.authUrl, "status", resp.Status(),
		"resp", resp.String(), "timeTaken", resp.Time().String())
	if resp.IsError() {
		return "", errors.New(resp.String())
	}
	timestamp := time.Now()
	grnt.timestamp = &timestamp

	return grnt.AccessToken, nil
}

func (c *EdaApiClient) getOauthBody(cred *clientCredentials, refreshToken string) map[string]string {
	oauthBody := make(map[string]string)
	oauthBody[KEY_CLIENT_ID] = cred.clientId
	oauthBody[KEY_CLIENT_SECRET] = cred.clientSecret
	if refreshToken != "" {
		oauthBody[KEY_GRANT_TYPE] = KEY_REFRESH_GRANT
		oauthBody[KEY_REFRESH_GRANT] = refreshToken
	} else {
		oauthBody[KEY_GRANT_TYPE] = KEY_PASSWORD_GRANT
		oauthBody[KEY_USERNAME] = cred.username
		oauthBody[KEY_PASSWORD] = cred.password
	}
	return oauthBody
}

func (c *EdaApiClient) getClientSecret(id string) (string, error) {
	keyCloakCred := &clientCredentials{
		authUrl:  fmt.Sprintf(OAUTH_URL, c.cfg.KcRealm),
		clientId: c.cfg.KcClientID,
		username: c.cfg.KcUsername,
		password: c.cfg.KcPassword,
	}
	accessToken, err := c.getAccessToken(keyCloakCred, c.keyCloakGrant)
	if err != nil {
		return "", err
	}

	result := []map[string]any{}
	resp, err := c.restClient.DoQuery(accessToken, CLIENT_URL, &result,
		map[string]string{"realm": c.cfg.EdaRealm},
		map[string]string{"clientId": id})
	if err != nil {
		return "", err
	}
	c.logger.Debug("getClientSecret()", "status", resp.Status(), "result", result)
	if len(result) == 0 {
		return "", fmt.Errorf("client not found: %s", id)
	}
	secret, ok := result[0]["secret"]
	if !ok {
		return "", fmt.Errorf("client secret not found for client: %s", id)
	}
	c.logger.Debug("getClientSecret()", "secret", secret)
	return secret.(string), nil
}

func (c *EdaApiClient) Create(ctx context.Context, pathUrl string, pathParams map[string]string, body any, result any) error {
	return c.Execute(ctx, pathUrl, rest.HTTP_POST, pathParams, nil, body, result)
}

func (c *EdaApiClient) Get(ctx context.Context, pathUrl string, pathParams map[string]string, result any) error {
	return c.Execute(ctx, pathUrl, rest.HTTP_GET, pathParams, nil, nil, result)
}

func (c *EdaApiClient) GetByQuery(ctx context.Context, pathUrl string, pathParams, queryParams map[string]string, result any) error {
	return c.Execute(ctx, pathUrl, rest.HTTP_GET, pathParams, queryParams, nil, result)
}

func (c *EdaApiClient) Update(ctx context.Context, pathUrl string, pathParams map[string]string, body any, result any) error {
	return c.Execute(ctx, pathUrl, rest.HTTP_PUT, pathParams, nil, body, result)
}

func (c *EdaApiClient) Delete(ctx context.Context, pathUrl string, pathParams map[string]string, result any) error {
	return c.Execute(ctx, pathUrl, rest.HTTP_DELETE, pathParams, nil, nil, result)
}

func (c *EdaApiClient) Execute(ctx context.Context, pathUrl, method string,
	pathParams, queryParams map[string]string, body, result any) error {
	accessToken, err := c.getEdaAccessToken()
	if err != nil {
		return err
	}
	resp, err := c.restClient.DoExecute(method, pathUrl, accessToken, body, result, pathParams, queryParams, nil)
	if err != nil {
		return err
	}
	c.logger.Info("execute()::"+method+" "+pathUrl,
		"pathParams", pathParams,
		"queryParams", queryParams,
		"status", resp.Status(),
		"timeTaken", resp.Time().String(),
	)
	if resp.IsError() {
		return fmt.Errorf(resp.Status(), resp.String())
	}
	return nil
}
