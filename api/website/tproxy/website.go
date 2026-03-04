package tproxy

import (
	"safeline/api"
	"safeline/api/website"
)

type API struct {
	*website.API
}

func New(baseUrl, token string) *API {
	return &API{
		API: &website.API{API: *api.New(baseUrl, token, "/api/HardwareTransparentProxyWebsiteAPI")},
	}
}

func NewFromAPI(a *api.API) *API {
	a.URI = "/api/HardwareTransparentProxyWebsiteAPI"
	return &API{
		API: &website.API{API: *a},
	}
}

type Data struct {
	// POST
	AssetGroup  int        `json:"asset_group,omitempty"` // 23+
	IsEnabled   bool       `json:"is_enabled"`            // 21+
	Name        string     `json:"name"`
	ServerNames []string   `json:"server_names"`
	Addrs       []Addrs    `json:"addrs"`
	SslCert     int        `json:"ssl_cert,omitempty"`
	Interface   string     `json:"interface"`
	PolicyGroup int        `json:"policy_group"`
	Remark      string     `json:"remark,omitempty"`
	URLPaths    []URLPaths `json:"url_paths,omitempty"` // 23+

	// PUT
	DetectorIPSource []string `json:"detector_ip_source"`
	ProxyIPList      []string `json:"proxy_ip_list"`
	ProxyIPGroups    []int    `json:"proxy_ip_groups"`

	IPSource *website.IPSource `json:"-"`
}
type Addrs struct {
	IpPort  string `json:"ip_port"`
	Ssl     bool   `json:"ssl,omitempty"`
	Sni     bool   `json:"sni,omitempty"`      // 21+
	HTTP2   bool   `json:"http2,omitempty"`    // 23+
	NonHTTP bool   `json:"non_http,omitempty"` // 23+
}
type URLPaths struct {
	Op      string `json:"op"`
	URLPath string `json:"url_path"`
}

func (cli *API) Create(data *Data) ([]byte, error) {
	return cli.Post(data)
}

// ConfigOption 嵌套了 Data 结构体
// 作用一是反序列化站点响应的数据，用于导出站点数据为 csv 模版；
// 二是为批量更新站点字段时解析 csv 模版到此结构体
type ConfigOption struct {
	Data
	Id        int       `json:"id"`
	AccessLog AccessLog `json:"access_log,omitempty"`
}

type AccessLog struct {
	IsEnabled bool   `json:"is_enabled"`
	LogOption string `json:"log_option,omitempty"`
	ReqBody   bool   `json:"req_body,omitempty"`
	RspBody   bool   `json:"rsp_body,omitempty"`
}
