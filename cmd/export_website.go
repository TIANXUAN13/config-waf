package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	urlpkg "net/url"
	"os"

	"safeline/api"
	"safeline/api/website/hproxy"
	"safeline/api/website/tproxy"
)

func openFile(cli *api.API, filename string) (*os.File, *csv.Writer) {
	u, _ := urlpkg.Parse(cli.BaseUrl)
	filename = fmt.Sprintf("%s_%s.csv", filename, u.Hostname())
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0666)
	panicIf(err)
	//defer f.Close()
	// utf-8 bom
	_, _ = f.WriteString("\xEF\xBB\xBF")
	w := csv.NewWriter(f)
	w.UseCRLF = true
	return f, w
}

func exportTransparentBridgeWebsite(cli *api.API, filename, mode string) {
	f, w := openFile(cli, filename)
	defer f.Close()
	defer w.Flush()
	headers := [][]string{
		{"资产组ID", "asset_group"},
		{"站点是否启用", "is_enabled"},
		{"防护站点名称", "name"},
		{"服务器地址", "addrs"},
		{"绑定域名", "server_names"},
		{"防护策略ID", "policy_group"},
		{"备注信息", "remark"},
		{"源IP获取方式", "detector_ip_source"},
		{"代理IP或者IP组ID", "proxy_ip_list_or_proxy_group"},
	}
	var h1, h2 []string
	for _, v := range headers {
		h1 = append(h1, v[0])
		h2 = append(h2, v[1])
	}
	panicIf(w.Write(h1))
	panicIf(w.Write(h2))

	b, err := getWebsite(cli, mode)
	panicIf(err)
	rsp := &TransparentBridgWebsiteResp{}
	panicIf(json.Unmarshal(b, rsp))
	var data [][]string
	length := len(rsp.Data)
	for i := length - 1; i >= 0; i-- {
		v := rsp.Data[i]
		data = append(data, []string{
			intToString(v.AssetGroup),
			boolToString(v.IsEnabled),
			v.Name,
			listToString(v.Addrs),
			listToString(v.ServerNames),
			intToString(v.PolicyGroup),
			v.Remark,
			listToString(v.DetectorIPSource),
			func(v1 []string, v2 []int) string {
				s1 := listToString(v1)
				if s1 != "" {
					return s1
				}
				return intListToString(v2)
			}(v.ProxyIPList, v.ProxyIPGroups),
		})
	}
	panicIf(w.WriteAll(data))
}

func exportTransparentProxyWebsite(cli *api.API, filename, mode string) {
	f, w := openFile(cli, filename)
	defer f.Close()
	defer w.Flush()
	headers := [][]string{
		{"资产组ID", "asset_group"},
		{"站点是否启用", "is_enabled"},
		{"防护站点名称", "name"},
		{"服务器地址", "addrs"},
		{"绑定域名", "server_names"},
		{"防护策略ID", "policy_group"},
		{"工作组名", "interface"},
		{"TCP转发", "non_http"},
		{"HTTPS站点", "ssl"},
		{"SSL证书ID", "ssl_cert"},
		{"HTTP2", "http2"},
		{"SNI转发", "sni"},
		{"备注信息", "remark"},
		{"源IP获取方式", "detector_ip_source"},
		{"代理IP或者IP组ID", "proxy_ip_list_or_proxy_group"},
		{"访问日志设置", "access_log"},
	}
	var h1, h2 []string
	for _, v := range headers {
		h1 = append(h1, v[0])
		h2 = append(h2, v[1])
	}
	panicIf(w.Write(h1))
	panicIf(w.Write(h2))

	b, err := getWebsite(cli, mode)
	panicIf(err)
	rsp := &TransparentProxyWebsiteResp{}
	panicIf(json.Unmarshal(b, rsp))
	var data [][]string
	length := len(rsp.Data)
	dumpJson := func(v interface{}) string {
		b, err := json.Marshal(v)
		panicIf(err)
		return string(b)
	}
	for i := length - 1; i >= 0; i-- {
		v := rsp.Data[i]
		var nonHttp, isSsl, isSni, isHttp2 bool
		data = append(data, []string{
			intToString(v.AssetGroup),
			boolToString(v.IsEnabled),
			v.Name,
			func(addrs []tproxy.Addrs) string {
				sl := make([]string, 0)
				for _, addr := range addrs {
					isSni = addr.Sni
					isSsl = addr.Ssl
					sl = append(sl, addr.IpPort)
				}
				return listToString(sl)
			}(v.Addrs),
			listToString(v.ServerNames),
			intToString(v.PolicyGroup),
			v.Interface,
			boolToString(nonHttp),
			boolToString(isSsl),
			intToString(v.SslCert),
			boolToString(isHttp2),
			boolToString(isSni),
			v.Remark,
			listToString(v.DetectorIPSource),
			func(v1 []string, v2 []int) string {
				s1 := listToString(v1)
				if s1 != "" {
					return s1
				}
				return intListToString(v2)
			}(v.ProxyIPList, v.ProxyIPGroups),
			dumpJson(v.AccessLog),
		})
	}
	panicIf(w.WriteAll(data))
}

func exportHardwareReverseProxyWebsite(cli *api.API, filename, mode string) {
	f, w := openFile(cli, filename)
	defer f.Close()
	defer w.Flush()
	headers := [][]string{
		{"资产组ID", "asset_group"},
		{"站点是否启用", "is_enabled"},
		{"防护站点名称", "name"},
		{"绑定域名", "server_names"},
		{"绑定端口", "ports"},
		{"TCP转发", "non_http"},
		{"HTTPS站点", "ssl"},
		{"SSL证书ID", "ssl_cert"},
		{"HTTP2", "http2"},
		{"SNI转发", "sni"},
		{"后端Server", "backend_config"},
		{"工作组名", "interface"},
		{"工作组IP", "ip"},
		{"防护策略ID", "policy_group"},
		{"备注信息", "remark"},
		{"源IP获取方式", "detector_ip_source"},
		{"代理IP或者IP组ID", "proxy_ip_list_or_proxy_group"},
		{"生效URL路径", "url_paths"},
		{"生效检测节点", "selected_tengine"},
		{"用户识别方式", "session_method"},
		{"回源IP配置", "proxy_bind_config"},
		{"访问日志设置", "access_log"},
		{"HTTP头设置", "header_config"},
		{"保持连接配置", "keepalive_config"},
		{"业务服务器获取源IP XFF 配置", "x_forwarded_for_action"},
	}
	var h1, h2 []string
	for _, v := range headers {
		h1 = append(h1, v[0])
		h2 = append(h2, v[1])
	}
	panicIf(w.Write(h1))
	panicIf(w.Write(h2))

	b, err := getWebsite(cli, mode)
	panicIf(err)
	rsp := &HardwareReverseProxyWebsiteResp{}
	panicIf(json.Unmarshal(b, rsp))
	var data [][]string
	length := len(rsp.Data)
	dumpJson := func(v interface{}) string {
		b, err := json.Marshal(v)
		panicIf(err)
		return string(b)
	}
	for i := length - 1; i >= 0; i-- {
		v := rsp.Data[i]
		if v.BackendConfig.Type != "proxy" {
			fmt.Printf("warning: 站点:{%s} 响应方式方式为:{%s} (只支持导出响应方式为代理的站点)\n", v.Name, v.BackendConfig.Type)
			continue
		}
		var nonHttp, isSsl, isSni, isHttp2 bool
		data = append(data, []string{
			intToString(v.AssetGroup),
			boolToString(v.IsEnabled),
			v.Name,
			listToString(v.ServerNames),
			func(ports []hproxy.Ports) string {
				sl := make([]string, 0)
				for _, port := range ports {
					nonHttp = port.NonHTTP
					isSni = port.Sni
					isSsl = port.Ssl
					isHttp2 = port.HTTP2
					sl = append(sl, intToString(port.Port))
				}
				return listToString(sl)
			}(v.Ports),
			boolToString(nonHttp),
			boolToString(isSsl),
			intToString(v.SslCert),
			boolToString(isHttp2),
			boolToString(isSni),
			func(config hproxy.BackendConfig) string {
				sl := make([]string, 0)
				sl = append(sl, config.LoadBalancePolicy)

				if config.LoadBalancePolicy == "Session Sticky" {
					sl = append(sl, intToString(config.SessionStickyCookieMaxAge))
				}
				for _, sv := range config.Servers {
					url := fmt.Sprintf("%s://%s:%d", sv.Protocol, sv.Host, sv.Port)
					if sv.Weight > 1 && config.LoadBalancePolicy == "Round Robin" {
						url = fmt.Sprintf("%s://%s:%d %d", sv.Protocol, sv.Host, sv.Port, sv.Weight)
					}
					sl = append(sl, url)
				}
				return listToString(sl)
			}(v.BackendConfig),
			v.Interface,
			listToString(v.IP),
			intToString(v.PolicyGroup),
			v.Remark,
			listToString(v.DetectorIPSource),
			func(v1 []string, v2 []int) string {
				s1 := listToString(v1)
				if s1 != "" {
					return s1
				}
				return intListToString(v2)
			}(v.ProxyIPList, v.ProxyIPGroups),
			dumpJson(v.URLPaths),
			dumpJson(v.SelectedTengine),
			dumpJson(v.SessionMethod),
			dumpJson(v.ProxyBindConfig),
			dumpJson(v.AccessLog),
			dumpJson(v.BackendConfig.HeaderConfig),
			v.BackendConfig.KeepaliveConfig,
			v.BackendConfig.XForwardedForAction,
		})
	}
	panicIf(w.WriteAll(data))
}

func ExportWebsite(cli *api.API, filename, mode string) {
	switch mode {
	case "TransparentBridge":
		exportTransparentBridgeWebsite(cli, filename, mode)
	case "TransparentProxy":
		exportTransparentProxyWebsite(cli, filename, mode)
	case "HardwareReverseProxy":
		exportHardwareReverseProxyWebsite(cli, filename, mode)
	case "SoftwareReverseProxy":
		exportHardwareReverseProxyWebsite(cli, filename, mode)
	default:
	}
}
