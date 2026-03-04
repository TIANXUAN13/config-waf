package rule

import (
	urlpkg "net/url"
	"safeline/api"
)

type API struct {
	api.API
}

func New(baseUrl, token string) *API {
	return &API{
		*api.New(baseUrl, token, "/api/PolicyRuleAPI"),
	}
}

func NewFromAPI(a *api.API) *API {
	a.URI = "/api/PolicyRuleAPI"
	return &API{
		API: *a,
	}
}

type Data struct {
	ID                  int           `json:"id"`
	Version             string        `json:"version"`
	Websites            []int         `json:"websites"`
	Target              string        `json:"target"`
	Pattern             interface{}   `json:"pattern"`
	Action              string        `json:"action"`
	IsEnabled           bool          `json:"is_enabled"`
	IsExpired           bool          `json:"is_expired"`
	Comment             string        `json:"comment"`
	AttackType          int           `json:"attack_type"`
	LogOption           string        `json:"log_option"`
	CreateTime          string        `json:"create_time"`
	LastUpdateTime      string        `json:"last_update_time"`
	ExpireTime          *string       `json:"expire_time"`
	IsGlobal            bool          `json:"is_global"`
	ForbiddenPageConfig interface{}   `json:"forbidden_page_config"`
	Priority            int           `json:"priority"`
	ModuleManagement    interface{}   `json:"module_management"`
	RiskLevel           int           `json:"risk_level"`
	ModulesList         []interface{} `json:"modules_list"`
}

func (cli *API) Fetch(query string) ([]byte, error) {
	q := urlpkg.Values{}
	if query != "" {
		q, _ = urlpkg.ParseQuery(query)
	}
	return cli.Get(q)
}

func (cli API) Create(v *Data) ([]byte, error) {
	if v.Action == "modify_module" {
		cli.URI = "/api/ModifyModulePolicyRuleAPI"
	}
	return cli.Post(v)
}
