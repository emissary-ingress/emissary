package v1

type FilterSpec struct {
	AmbassadorID AmbassadorID  `json:"ambassador_id"`
	OAuth2       *FilterOAuth2 `json:",omitempty"`
	Plugin       *FilterPlugin `json:",omitempty"`
}
