package desktop

// ServiceSettings is the 服务 page.
type ServiceSettings struct {
	ListenMode            string   `json:"listenMode"`
	CustomHost            string   `json:"customHost"`
	Port                  int      `json:"port"`
	ClientAPIKeys         []string `json:"clientApiKeys"`
	ProxyURL              string   `json:"proxyUrl"`
	RoutingStrategy       string   `json:"routingStrategy"`
	Debug                 bool     `json:"debug"`
}

// KeyDraft is one upstream API key edited on a provider page.
type KeyDraft struct {
	APIKey  string `json:"apiKey"`
	BaseURL string `json:"baseUrl"`
	Prefix  string `json:"prefix"`
}

// ModelDraft is one client-visible model on an OpenAI-compatible provider.
type ModelDraft struct {
	Name  string `json:"name"`
	Alias string `json:"alias"`
}

// OpenAIDraft is one OpenAI-compatible upstream.
type OpenAIDraft struct {
	Name    string       `json:"name"`
	BaseURL string       `json:"baseUrl"`
	APIKeys []string     `json:"apiKeys"`
	Models  []ModelDraft `json:"models"`
}

// Account is a browser-login record safe to show in the window.
type Account struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Label    string `json:"label"`
}

// Status is the navbar control and the address clients should use.
type Status struct {
	Running         bool   `json:"running"`
	Action          string `json:"action"`
	Address         string `json:"address"`
	AllInterfaces   bool   `json:"allInterfaces"`
	RestartRequired bool   `json:"restartRequired"`
	SavedAddress    string `json:"savedAddress"`
	Error           string `json:"error"`
	LoginActive     bool   `json:"loginActive"`
}

// Listen is the host, port, and TLS actually bound or saved.
type Listen struct {
	Host       string
	Port       int
	TLSEnabled bool
	TLSCert    string
	TLSKey     string
}
