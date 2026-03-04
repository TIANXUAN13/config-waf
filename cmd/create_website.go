package main

import (
	"encoding/json"
	"fmt"
	urlpkg "net/url"
	"os"
	"reflect"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"safeline/api"
	"safeline/api/website"
	"safeline/api/website/hproxy"
	"safeline/api/website/tbridge"
	"safeline/api/website/tproxy"
)

var DEBUG bool

func CreateTransparentBridgeWebsite(cli *tbridge.API, data *tbridge.Data) (b []byte, err error) {
	b, err = cli.Create(data)
	if ok, e := api.OK2(b, err); !ok {
		err = fmt.Errorf("failed to init website: %s", e)
		return
	}
	if len(data.IPSource.DetectorIPSource) <= 1 {
		return
	}

	if b, err = updateXffData(b, data.IPSource); err != nil {
		err = fmt.Errorf("failed to update ip source:%s %s", data.IPSource.DetectorIPSource, err)
		return
	}
	result := gjson.GetBytes(b, "data")
	b, err = cli.Update([]byte(result.Raw))
	if ok, e := api.OK2(b, err); !ok {
		err = fmt.Errorf("failed to update ip source:%s %s", data.IPSource.DetectorIPSource, e)
		return
	}
	return
}

func updateXffData(data []byte, ip *website.IPSource) (b []byte, err error) {
	b, err = sjson.SetBytes(data, "data.detector_ip_source", ip.DetectorIPSource)
	b, err = sjson.SetBytes(b, "data.proxy_ip_list", ip.ProxyIPList)
	b, err = sjson.SetBytes(b, "data.proxy_ip_groups", ip.ProxyIPGroups)
	if err != nil {
		err = fmt.Errorf("sjson.SetBytes: %s", err)
	}
	return
}

func GenTransparentBridgeData(filename string) []*tbridge.Data {
	records, _, H := loadTemplate(filename)
	ret := make([]*tbridge.Data, 0, len(records))
	for idx, record := range records {
		data := parseTransparentBridgeCsvRow(record, H)
		if DEBUG {
			jsonBytes, _ := json.Marshal(data)
			fmt.Printf("row:%d websitename:%s csv data => \n%s\n", idx+3, data.Name, jsonBytes)
		}
		ret = append(ret, data)
	}
	return ret
}

func parseTransparentBridgeCsvRow(record []string, H map[string]int) *tbridge.Data {
	ret := &tbridge.Data{}
	ret.AssetGroup = stringToInt(record[H["asset_group"]])
	ret.IsEnabled = stringToBool(record[H["is_enabled"]])
	ret.Name = strip(record[H["name"]])
	ret.Addrs = stringToList(record[H["addrs"]])
	ret.ServerNames = stringToList(record[H["server_names"]])
	ret.PolicyGroup = stringToInt(record[H["policy_group"]])
	ret.Remark = strip(record[H["remark"]])
	ret.IPSource = genIPSource(record[H["detector_ip_source"]], record[H["proxy_ip_list_or_proxy_group"]])
	return ret
}

func genIPSource(source, proxyIp string) *website.IPSource {
	ret := &website.IPSource{
		DetectorIPSource: []string{"Socket"},
		ProxyIPList:      []string{},
		ProxyIPGroups:    []int{},
	}

	sc := stringToList(source)
	if len(sc) > 1 {
		ret.DetectorIPSource = sc
	}
	for _, v := range sc {
		if v == "rightmost_non_proxy_ip" {
			ips := stringToList(proxyIp)
			if strings.Contains(proxyIp, ".") {
				// ip 列表
				ret.ProxyIPList = append(ret.ProxyIPList, ips...)
			} else {
				// ip 组 id 列表
				var ipGroup []int
				for _, groupId := range ips {
					_id := stringToInt(groupId)
					ipGroup = append(ipGroup, _id)
				}
				ret.ProxyIPGroups = append(ret.ProxyIPGroups, ipGroup...)
			}
			break
		}
	}
	return ret
}

func CreateHardwareReverseProxyWebsite(cli *hproxy.API, data *hproxy.Data) (b []byte, err error) {
	b, err = cli.Create(data)
	if ok, e := api.OK2(b, err); !ok {
		err = fmt.Errorf("failed to init website: %s", e)
		return
	}
	if len(data.IPSource.DetectorIPSource) <= 1 {
		return
	}

	if b, err = updateXffData(b, data.IPSource); err != nil {
		err = fmt.Errorf("failed to update ip source:%s %s", data.IPSource.DetectorIPSource, err)
		return
	}
	result := gjson.GetBytes(b, "data")
	b, err = cli.Update([]byte(result.Raw))
	if ok, e := api.OK2(b, err); !ok {
		err = fmt.Errorf("failed to update ip source:%s %s", data.IPSource.DetectorIPSource, e)
		return
	}
	return
}

func GenHardwareReverseProxyData(filename string) []*hproxy.Data {
	records, _, H := loadTemplate(filename)
	ret := make([]*hproxy.Data, 0, len(records))
	for idx, record := range records {
		data := parseHardwareReverseProxyCsvRow(record, H)
		if DEBUG {
			jsonBytes, _ := json.Marshal(data)
			fmt.Printf("row:%d websitename:%s csv data => \n%s\n", idx+3, data.Name, jsonBytes)
		}
		ret = append(ret, data)
	}

	return ret
}

func parseHardwareReverseProxyCsvRow(record []string, H map[string]int) *hproxy.Data {
	ret := &hproxy.Data{}
	ret.SessionMethod = hproxy.SessionMethod{Type: "off"}
	ret.SelectedTengine = hproxy.SelectedTengine{Type: "all"}
	ret.AssetGroup = stringToInt(record[H["asset_group"]])
	ret.IsEnabled = stringToBool(record[H["is_enabled"]])
	ret.Name = strip(record[H["name"]])
	ret.ServerNames = stringToList(record[H["server_names"]])
	ports := stringToList(record[H["ports"]])
	ret.Ports = genPorts(ports, record[H["non_http"]], record[H["ssl"]], record[H["http2"]], record[H["sni"]])
	if stringToBool(record[H["ssl"]]) {
		ret.SslCert = stringToInt(record[H["ssl_cert"]])
	}
	ret.BackendConfig = genBackendConfig(stringToList(record[H["backend_config"]]))
	ret.Interface = strip(record[H["interface"]])
	ret.IP = stringToList(record[H["ip"]])
	ret.PolicyGroup = stringToInt(record[H["policy_group"]])
	ret.Remark = strip(record[H["remark"]])
	ret.IPSource = genIPSource(record[H["detector_ip_source"]], record[H["proxy_ip_list_or_proxy_group"]])
	return ret
}

func genPorts(ports []string, tcp, ssl, http2, sni string) []hproxy.Ports {
	var ret []hproxy.Ports

	nonHttp := stringToBool(tcp)
	isSsl := stringToBool(ssl)
	isHttp2 := stringToBool(http2)
	isSni := stringToBool(sni)

	for _, p := range ports {
		port := stringToInt(p)
		ret = append(ret, hproxy.Ports{Port: port, NonHTTP: nonHttp, Ssl: isSsl, HTTP2: isHttp2, Sni: isSni})
	}
	return ret
}

func genBackendConfig(config []string) hproxy.BackendConfig {
	var ret hproxy.BackendConfig
	ret.Type = "proxy"
	ret.XForwardedForAction = "append"
	ret.LoadBalancePolicy = config[0]

	var svs []hproxy.Servers
	if ret.LoadBalancePolicy == "Round Robin" {
		for _, cfg := range config[1:] {
			urlWeights := strings.Split(strip(cfg), " ")
			url := urlWeights[0]
			w := 1
			if len(urlWeights) > 1 {
				idx := len(urlWeights) - 1
				w = stringToInt(strip(urlWeights[idx]))
			}
			url = strip(url)
			u, _ := urlpkg.Parse(url)
			s := hproxy.Servers{Protocol: u.Scheme, Host: u.Hostname(), Port: stringToInt(u.Port()), Weight: w}
			svs = append(svs, s)
		}
		ret.Servers = svs
		return ret
	}

	switch ret.LoadBalancePolicy {
	case "Session Sticky":
		ret.SessionStickyCookieMaxAge = stringToInt(config[1])
		config = config[2:]
	case "Least Connected", "Hash", "IP Hash":
		config = config[1:]
	}
	for _, cfg := range config {
		u, _ := urlpkg.Parse(cfg)
		s := hproxy.Servers{Protocol: u.Scheme, Host: u.Hostname(), Port: stringToInt(u.Port())}
		svs = append(svs, s)
	}

	ret.Servers = svs
	return ret
}

func CreateTransparentProxyWebsite(cli *tproxy.API, data *tproxy.Data) (b []byte, err error) {
	b, err = cli.Create(data)
	if ok, e := api.OK2(b, err); !ok {
		err = fmt.Errorf("failed to init website: %s", e)
		return
	}
	if len(data.IPSource.DetectorIPSource) <= 1 {
		return
	}

	if b, err = updateXffData(b, data.IPSource); err != nil {
		err = fmt.Errorf("failed to update ip source:%s %s", data.IPSource.DetectorIPSource, err)
		return
	}
	result := gjson.GetBytes(b, "data")
	b, err = cli.Update([]byte(result.Raw))
	if ok, e := api.OK2(b, err); !ok {
		err = fmt.Errorf("failed to update ip source:%s %s", data.IPSource.DetectorIPSource, e)
		return
	}
	return
}

func GenTransparentProxyData(filename string) []*tproxy.Data {
	records, _, H := loadTemplate(filename)
	ret := make([]*tproxy.Data, 0, len(records))
	for idx, record := range records {
		data := parseTransparentProxyCsvRow(record, H)
		if DEBUG {
			jsonBytes, _ := json.Marshal(data)
			fmt.Printf("row:%d websitename:%s csv data => \n%s\n", idx+3, data.Name, jsonBytes)
		}
		ret = append(ret, data)
	}
	return ret
}

func parseTransparentProxyCsvRow(record []string, H map[string]int) *tproxy.Data {
	ret := &tproxy.Data{}
	ret.AssetGroup = stringToInt(record[H["asset_group"]])
	ret.IsEnabled = stringToBool(record[H["is_enabled"]])
	ret.Name = strip(record[H["name"]])
	ret.Addrs = genAddrs(
		stringToList(record[H["addrs"]]),
		record[H["non_http"]],
		record[H["ssl"]],
		record[H["http2"]],
		record[H["sni"]],
	)
	ret.ServerNames = stringToList(record[H["server_names"]])
	ret.PolicyGroup = stringToInt(record[H["policy_group"]])
	ret.Interface = strip(record[H["interface"]])
	if stringToBool(record[H["ssl"]]) {
		ret.SslCert = stringToInt(record[H["ssl_cert"]])
	}
	ret.Remark = strip(record[H["remark"]])
	ret.IPSource = genIPSource(record[H["detector_ip_source"]], record[H["proxy_ip_list_or_proxy_group"]])
	return ret
}

func genAddrs(ipPorts []string, tcp, ssl, http2, sni string) []tproxy.Addrs {
	ret := make([]tproxy.Addrs, 0)

	nonHttp := stringToBool(tcp)
	isSsl := stringToBool(ssl)
	isHttp2 := stringToBool(http2)
	isSni := stringToBool(sni)

	for _, ipPort := range ipPorts {
		ipPort := strip(ipPort)
		ret = append(ret, tproxy.Addrs{IpPort: ipPort, NonHTTP: nonHttp, Ssl: isSsl, HTTP2: isHttp2, Sni: isSni})
	}
	return ret
}

func CreateWebsite(a *api.API, filename string, mode string) {
	lengthCh := make(chan int)
	indexCh := make(chan int)
	stopCh := make(chan struct{})
	go progressBar(lengthCh, indexCh, stopCh)

	switch mode {
	case "TransparentBridge":
		cli := tbridge.NewFromAPI(a)
		rows := GenTransparentBridgeData(filename)
		for idx, line := range rows {
			_, err := CreateTransparentBridgeWebsite(cli, line)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[error] row:%d websitename:%s %+v\n", idx+3, line.Name, err)
			}
			lengthCh <- len(rows)
			indexCh <- idx
		}
	case "TransparentProxy":
		cli := tproxy.NewFromAPI(a)
		rows := GenTransparentProxyData(filename)
		for idx, line := range rows {
			_, err := CreateTransparentProxyWebsite(cli, line)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[error] row:%d websitename:%s %+v\n", idx+3, line.Name, err)
			}
			lengthCh <- len(rows)
			indexCh <- idx
		}
	case "HardwareReverseProxy":
		cli := hproxy.NewFromAPI(a)
		rows := GenHardwareReverseProxyData(filename)
		for idx, line := range rows {
			_, err := CreateHardwareReverseProxyWebsite(cli, line)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[error] row:%d websitename:%s %+v\n", idx+3, line.Name, err)
			}
			lengthCh <- len(rows)
			indexCh <- idx
		}
	case "SoftwareReverseProxy":
		cli := hproxy.NewFromAPI(a)
		cli.URI = "/api/SoftwareReverseProxyWebsiteAPI"
		rows := GenHardwareReverseProxyData(filename)
		for idx, line := range rows {
			line.Interface = ""
			if reflect.DeepEqual(line.IP, []string{""}) || reflect.DeepEqual(line.IP, []string{"::"}) {
				// 在 23.01.011 之前版本，软件雷池监听的 ip 无法配置，给出的 csv 模版填的是 ::
				// 无论 csv 模版填的是什么，在这里都会替换成空值
				// 为空值的话，雷池后台会同时监听 0.0.0.0 与 ::
				line.IP = []string{}
			}
			_, err := CreateHardwareReverseProxyWebsite(cli, line)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[error] row:%d websitename:%s %+v\n", idx+3, line.Name, err)
			}
			lengthCh <- len(rows)
			indexCh <- idx
		}
	default:
		fmt.Fprintf(os.Stderr, "[error] %s mode unsupported\n", mode)
	}

	fmt.Println()
	stopCh <- struct{}{}
}
