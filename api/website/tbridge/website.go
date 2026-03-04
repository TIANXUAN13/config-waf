package tbridge

import (
	"safeline/api"
	"safeline/api/website"
)

type API struct {
	*website.API
}

func New(baseUrl, token string) *API {
	return &API{
		API: &website.API{API: *api.New(baseUrl, token, "/api/HardwareTransparentBridgingWebsiteAPI")},
	}
}

func NewFromAPI(a *api.API) *API {
	a.URI = "/api/HardwareTransparentBridgingWebsiteAPI"
	return &API{
		API: &website.API{API: *a},
	}
}

type Data struct {
	// POST
	AssetGroup  int        `json:"asset_group,omitempty"` // 23+
	IsEnabled   bool       `json:"is_enabled"`            // 21+
	Name        string     `json:"name"`
	Addrs       []string   `json:"addrs"`
	ServerNames []string   `json:"server_names"`
	PolicyGroup int        `json:"policy_group"`
	Remark      string     `json:"remark,omitempty"`
	URLPaths    []URLPaths `json:"url_paths,omitempty"` // 23+

	// PUT
	DetectorIPSource []string `json:"detector_ip_source"`
	ProxyIPList      []string `json:"proxy_ip_list"`
	ProxyIPGroups    []int    `json:"proxy_ip_groups"`

	Id       int               `json:"id,omitempty"`
	IPSource *website.IPSource `json:"-"`
}
type URLPaths struct {
	Op      string `json:"op"`
	URLPath string `json:"url_path"`
}

func (cli *API) Create(data *Data) ([]byte, error) {
	return cli.Post(data)
}
