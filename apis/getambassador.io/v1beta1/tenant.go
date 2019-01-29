package v1

type TenantSpec struct {
	Tenants []TenantObject `json:"tenants"`
}

type TenantObject struct {
	CallbackURL string `json:"-"` // is calculated from TenantURL
	TenantURL   string `json:"tenantUrl"`
	TLS         bool   `json:"tls"`
	Domain      string `json:"-"` // is calculated from TenantURL
	Audience    string `json:"audience"`
	ClientID    string `json:"clientId"`
	Secret      string `json:"secret"`
}
