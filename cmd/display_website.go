package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"

	"safeline/api"
	"safeline/api/website/hproxy"
	"safeline/api/website/tbridge"
	"safeline/api/website/tproxy"
)

type TransparentBridgWebsiteResp struct {
	Err  interface{}    `json:"err"`
	Data []tbridge.Data `json:"data"`
	Msg  interface{}    `json:"msg"`
}

type TransparentProxyWebsiteResp struct {
	Err  interface{}           `json:"err"`
	Data []tproxy.ConfigOption `json:"data"`
	Msg  interface{}           `json:"msg"`
}

type HardwareReverseProxyWebsiteResp struct {
	Err  interface{}           `json:"err"`
	Data []hproxy.ConfigOption `json:"data"`
	Msg  interface{}           `json:"msg"`
}

func initPolicyMap(cli *api.API) map[int]string {
	b, err := getPolicyGroup(cli)
	if ok, err := api.OK2(b, err); !ok {
		panic(err)
	}

	group := &policyResp{}
	err = json.Unmarshal(b, group)
	if err != nil {
		panic(err)
	}

	ret := make(map[int]string)
	for _, v := range group.Data {
		ret[v.ID] = v.Name
	}
	return ret
}

func getWebsite(cli *api.API, mode string) ([]byte, error) {
	var uri string
	switch mode {
	case "TransparentBridge":
		uri = "/api/HardwareTransparentBridgingWebsiteAPI"
	case "TransparentProxy":
		uri = "/api/HardwareTransparentProxyWebsiteAPI"
	case "HardwareReverseProxy":
		uri = "/api/HardwareReverseProxyWebsiteAPI"
	case "SoftwareReverseProxy":
		uri = "/api/SoftwareReverseProxyWebsiteAPI"
	default:
		fmt.Printf("mode unsupported: %s\n", mode)
		os.Exit(1)
	}
	cli.URI = uri
	return cli.Get(nil)
}

func displayTransparentBridgeWebsite(b []byte, t table.Writer, policyMap map[int]string) {
	websites := &TransparentBridgWebsiteResp{}
	panicIf(json.Unmarshal(b, websites))
	t.AppendHeader(table.Row{"序号", "站点ID", "是否启用", "站点名", "服务器地址", "绑定域名", "防护策略", "源IP获取方式"})
	for idx, v := range websites.Data {
		t.AppendRow([]interface{}{
			idx + 1,
			v.Id,
			v.IsEnabled,
			v.Name,
			strings.Join(v.Addrs, "\n"),
			strings.Join(v.ServerNames, "\n"),
			func(group int) string {
				if group != 0 {
					return fmt.Sprintf("%s(id:%d)", policyMap[group], group)
				} else {
					return "不使用防护策略(id:0)"
				}

			}(v.PolicyGroup),
			strings.Join(v.DetectorIPSource, "\n"),
			//formatTime(v.CreateTime),
		})
		t.AppendSeparator()
	}
}

func displayTransparentProxyWebsite(b []byte, t table.Writer, policyMap map[int]string) {
	websites := &TransparentProxyWebsiteResp{}
	panicIf(json.Unmarshal(b, websites))
	t.AppendHeader(table.Row{"序号", "站点ID", "是否启用", "站点名", "服务器地址", "绑定域名", "防护策略", "源IP获取方式"})
	for idx, v := range websites.Data {
		t.AppendRow([]interface{}{
			idx + 1,
			v.Id,
			v.IsEnabled,
			v.Name,
			func(addrs []tproxy.Addrs) string {
				sl := make([]string, 0)
				for _, addr := range addrs {
					sl = append(sl, addr.IpPort)
				}
				return strings.ReplaceAll(listToString(sl), ", ", "\n")
			}(v.Addrs),
			strings.Join(v.ServerNames, "\n"),
			func(group int) string {
				if group != 0 {
					return fmt.Sprintf("%s(id:%d)", policyMap[group], group)
				} else {
					return "不使用防护策略(id:0)"
				}
			}(v.PolicyGroup),
			strings.Join(v.DetectorIPSource, "\n"),
			//formatTime(v.CreateTime),
		})
		t.AppendSeparator()
	}
}

func displayHardwareReverseProxyWebsite(b []byte, t table.Writer, policyMap map[int]string) {
	websites := &HardwareReverseProxyWebsiteResp{}
	panicIf(json.Unmarshal(b, websites))
	t.AppendHeader(table.Row{"序号", "站点ID", "是否启用", "站点名", "监听端口", "绑定域名", "后端服务器地址", "防护策略", "源IP获取方式"})
	for idx, v := range websites.Data {
		t.AppendRow([]interface{}{
			idx + 1,
			v.Id,
			v.IsEnabled,
			v.Name,
			func(ports []hproxy.Ports) string {
				sl := make([]string, 0)
				for _, port := range ports {
					sl = append(sl, intToString(port.Port))
				}
				return strings.ReplaceAll(listToString(sl), ", ", "\n")
			}(v.Ports),
			strings.Join(v.ServerNames, "\n"),
			func(config hproxy.BackendConfig) string {
				if config.Type != "proxy" {
					return config.Type
				}
				sl := make([]string, 0)
				for _, sv := range config.Servers {
					url := fmt.Sprintf("%s://%s:%d", sv.Protocol, sv.Host, sv.Port)
					if sv.Weight != 1 {
						url = fmt.Sprintf("%s://%s:%d %d", sv.Protocol, sv.Host, sv.Port, sv.Weight)
					}
					sl = append(sl, url)
				}
				return strings.ReplaceAll(listToString(sl), ", ", "\n")
			}(v.BackendConfig),
			func(group int) string {
				if group != 0 {
					return fmt.Sprintf("%s(id:%d)", policyMap[group], group)
				} else {
					return "不使用防护策略(id:0)"
				}
			}(v.PolicyGroup),
			strings.Join(v.DetectorIPSource, "\n"),
			//formatTime(v.CreateTime),
		})
		t.AppendSeparator()
	}
}

func DisplayWebsite(cli *api.API, w *os.File, raw bool, args ...interface{}) {
	mode := args[0].(string)
	b, err := getWebsite(cli, mode)
	if ok, err := api.OK2(b, err); !ok {
		panic(err)
	}
	if raw {
		_, _ = w.Write(b)
		return
	}

	policyMap := initPolicyMap(cli)
	t := table.NewWriter()
	t.SetOutputMirror(w)
	switch mode {
	case "TransparentBridge":
		displayTransparentBridgeWebsite(b, t, policyMap)
	case "TransparentProxy":
		displayTransparentProxyWebsite(b, t, policyMap)
	case "HardwareReverseProxy", "SoftwareReverseProxy":
		displayHardwareReverseProxyWebsite(b, t, policyMap)
	default:
	}

	if w.Name() != os.Stdout.Name() {
		t.RenderCSV()
	} else {
		t.Render()
	}
}
