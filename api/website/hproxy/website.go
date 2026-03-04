package hproxy

import (
	"safeline/api"
	"safeline/api/website"
)

type API struct {
	*website.API
}

func New(baseUrl, token string) *API {
	return &API{
		API: &website.API{API: *api.New(baseUrl, token, "/api/HardwareReverseProxyWebsiteAPI")},
	}
}

func NewFromAPI(a *api.API) *API {
	a.URI = "/api/HardwareReverseProxyWebsiteAPI"
	return &API{
		API: &website.API{API: *a},
	}
}

// Data 结构体用于创建站点，序列化为原始 json
type Data struct {
	// POST
	AssetGroup      int             `json:"asset_group,omitempty"` // 23+
	IsEnabled       bool            `json:"is_enabled"`            // 21+
	Name            string          `json:"name"`
	ServerNames     []string        `json:"server_names"`
	Interface       string          `json:"interface"`
	IP              []string        `json:"ip"`
	Ports           []Ports         `json:"ports"`
	SslCert         int             `json:"ssl_cert,omitempty"`
	BackendConfig   BackendConfig   `json:"backend_config"`
	PolicyGroup     int             `json:"policy_group"`
	Remark          string          `json:"remark,omitempty"`
	SessionMethod   SessionMethod   `json:"session_method"`
	URLPaths        []URLPaths      `json:"url_paths,omitempty"`        // 23+
	SelectedTengine SelectedTengine `json:"selected_tengine,omitempty"` // 23+

	// PUT
	DetectorIPSource []string `json:"detector_ip_source"`
	ProxyIPList      []string `json:"proxy_ip_list"`
	ProxyIPGroups    []int    `json:"proxy_ip_groups"`

	IPSource *website.IPSource `json:"-"`
}
type Ports struct {
	Port    int  `json:"port"`
	Ssl     bool `json:"ssl"`
	HTTP2   bool `json:"http2"`
	Sni     bool `json:"sni,omitempty"`      // 21+
	NonHTTP bool `json:"non_http,omitempty"` // 23+
}
type Servers struct {
	Protocol  string `json:"protocol"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Weight    int    `json:"weight,omitempty"`
	IsEnabled bool   `json:"is_enabled,omitempty"` // 23+
}
type BackendConfig struct {
	Type                      string    `json:"type"`
	LoadBalancePolicy         string    `json:"load_balance_policy"`
	Servers                   []Servers `json:"servers"`
	XForwardedForAction       string    `json:"x_forwarded_for_action"`
	SessionStickyCookieMaxAge int       `json:"session_sticky_cookie_max_age,omitempty"`

	// PUT
	HeaderConfig     []HeaderConfig `json:"header_config,omitempty"`
	KeepaliveConfig  string         `json:"keepalive_config,omitempty"`
	KeepaliveTimeout int            `json:"keepalive_timeout,omitempty"`
	Keepalive        int            `json:"keepalive,omitempty"`
}
type SessionMethod struct {
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Pattern string `json:"pattern,omitempty"`
}
type HeaderConfig struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Action  string `json:"action"`
	Context string `json:"context"`
}
type URLPaths struct {
	Op      string `json:"op"`
	URLPath string `json:"url_path"`
}
type SelectedTengine struct {
	Type        string   `json:"type"`
	TengineList []string `json:"tengine_list,omitempty"`
}

func (cli *API) Create(data *Data) ([]byte, error) {
	return cli.Post(data)
}

// ConfigOption 嵌套了 Data 结构体
// 作用一是反序列化站点响应的数据，用于导出站点数据为 csv 模版；
// 二是为批量更新站点字段时解析 csv 模版到此结构体
type ConfigOption struct {
	Data
	Id              int             `json:"id"`
	ProxyBindConfig ProxyBindConfig `json:"proxy_bind_config"`
	AccessLog       AccessLog       `json:"access_log,omitempty"`
}
type ProxyBindConfig struct {
	Enable             bool        `json:"enable"`
	ProxyBindIPList    interface{} `json:"proxy_bind_ip_list,omitempty"`
	HashSelectIPMethod string      `json:"hash_select_ip_method"`
}

//	type ProxyBindIPList struct {
//		ID      string   `json:"id"`
//		IPGroup int      `json:"ip_group,omitempty"`
//		Ips     []string `json:"ips,omitempty"`
//	}

type AccessLog struct {
	IsEnabled bool   `json:"is_enabled"`
	LogOption string `json:"log_option,omitempty"`
	ReqBody   bool   `json:"req_body,omitempty"`
	RspBody   bool   `json:"rsp_body,omitempty"`
}
