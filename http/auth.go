package http

type Auth interface {
	GetCredentials() Auth
}

type BasicAuth struct {
	URL         string                 `json:"url"`
	Method      string                 `json:"method"`
	Username    string                 `json:"username"`
	Password    string                 `json:"password"`
	TokenField  string                 `json:"token_field"`
	ExpiryField string                 `json:"expiry_field"`
	Data        map[string]interface{} `json:"data"`
	Headers     map[string]string      `json:"headers"`
}

func (a *BasicAuth) GetCredentials() Auth {
	return a
}

type BearerToken struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

func (a *BearerToken) GetCredentials() Auth {
	return a
}

type OAuth2 struct {
	URL         string                 `json:"url"`
	Method      string                 `json:"method"`
	ClientID    string                 `json:"client_id"`
	Secret      string                 `json:"secret"`
	GrantType   string                 `json:"grant_type"`
	TokenField  string                 `json:"token_field"`
	ExpiryField string                 `json:"expiry_field"`
	Data        map[string]interface{} `json:"data"`
	Headers     map[string]string      `json:"headers"`
}

func (a *OAuth2) GetCredentials() Auth {
	return a
}
