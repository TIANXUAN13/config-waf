package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"safeline/api"
	"safeline/api/website/hproxy"
	"safeline/api/website/tbridge"
	"safeline/api/website/tproxy"
)

func parseHardwareReverseProxyExtraCsv(filename string) []*hproxy.ConfigOption {
	dataList := GenHardwareReverseProxyData(filename)
	records, _, H := loadTemplate(filename)
	ret := make([]*hproxy.ConfigOption, 0, len(records))
	for idx, data := range dataList {
		v := new(hproxy.ConfigOption)
		v.IPSource = data.IPSource // data.IPSource 忽略 json 序列化，信息会丢失
		b, err := json.Marshal(data)
		panicIf(err)
		panicIf(json.Unmarshal(b, v))

		row := records[idx]
		if strip(row[H["name"]]) != v.Name {
			panic("parse template error")
		}
		panicIf(json.Unmarshal([]byte(strip(row[H["url_paths"]])), &v.URLPaths))
		panicIf(json.Unmarshal([]byte(strip(row[H["selected_tengine"]])), &v.SelectedTengine))
		panicIf(json.Unmarshal([]byte(strip(row[H["session_method"]])), &v.SessionMethod))
		panicIf(json.Unmarshal([]byte(strip(row[H["proxy_bind_config"]])), &v.ProxyBindConfig))
		panicIf(json.Unmarshal([]byte(strip(row[H["access_log"]])), &v.AccessLog))
		panicIf(json.Unmarshal([]byte(strip(row[H["header_config"]])), &v.BackendConfig.HeaderConfig))
		v.BackendConfig.KeepaliveConfig = strip(row[H["keepalive_config"]])
		v.BackendConfig.XForwardedForAction = strip(row[H["x_forwarded_for_action"]])
		ret = append(ret, v)

		if DEBUG {
			jsonBytes, _ := json.Marshal(v)
			fmt.Printf("row:%d websitename:%s extra csv data => \n%s\n", idx+2, data.Name, jsonBytes)
		}
	}
	return ret
}

func parseTransparentProxyExtraCsv(filename string) []*tproxy.ConfigOption {
	dataList := GenTransparentProxyData(filename)
	records, _, H := loadTemplate(filename)
	ret := make([]*tproxy.ConfigOption, 0, len(records))
	for idx, data := range dataList {
		v := new(tproxy.ConfigOption)
		v.IPSource = data.IPSource // data.IPSource 忽略 json 序列化，信息会丢失
		b, err := json.Marshal(data)
		panicIf(err)
		panicIf(json.Unmarshal(b, v))

		row := records[idx]
		if strip(row[H["name"]]) != v.Name {
			panic("parse template error")
		}
		panicIf(json.Unmarshal([]byte(strip(row[H["access_log"]])), &v.AccessLog))
		ret = append(ret, v)

		if DEBUG {
			jsonBytes, _ := json.Marshal(v)
			fmt.Printf("row:%d websitename:%s extra csv data => \n%s\n", idx+2, data.Name, jsonBytes)
		}
	}
	return ret
}

func UpdateTransparentBridgeWebsite(cli *tbridge.API, data *tbridge.Data, limit ...int) (b []byte, err error) {
	ids, e := cli.GetIdByName(data.Name + "_" + strconv.Itoa(data.AssetGroup))
	if e != nil {
		return nil, fmt.Errorf("failed to GetIdByName(%s_%d): %s", data.Name, data.AssetGroup, e)
	} else if len(ids) < 1 {
		return nil, fmt.Errorf("failed to GetIdByName(%s_%d): %s", data.Name, data.AssetGroup, "not found")
	}

	for _, id := range ids {
		obj, e := cli.GetDetailById(id)
		if ok, e := api.OK2(obj, e); !ok {
			return nil, fmt.Errorf("failed to GetDetailById(%d): %s", id, e)
		}

		result := gjson.GetBytes(obj, "data.0")
		b = []byte(result.Raw)
		for _, n := range limit {
			switch n {
			case 0:
				b, err = sjson.SetBytes(b, "is_enabled", data.IsEnabled)
			case 1:
				b, err = sjson.SetBytes(b, "policy_group", data.PolicyGroup)
			case 2:
				b, err = sjson.SetBytes(b, "detector_ip_source", data.IPSource.DetectorIPSource)
				b, err = sjson.SetBytes(b, "proxy_ip_list", data.IPSource.ProxyIPList)
				b, err = sjson.SetBytes(b, "proxy_ip_groups", data.IPSource.ProxyIPGroups)
			case 5:
				b, err = sjson.SetBytes(b, "asset_group", data.AssetGroup)
			//case 0:
			//	b, err = sjson.SetBytes(b, "is_enabled", data.IsEnabled)
			//case 1:
			//	b, err = sjson.SetBytes(b, "name", data.Name)
			//case 2:
			//	b, err = sjson.SetBytes(b, "addrs", data.Addrs)
			//case 3:
			//	b, err = sjson.SetBytes(b, "server_names", data.ServerNames)
			//case 4:
			//	b, err = sjson.SetBytes(b, "policy_group", data.PolicyGroup)
			//case 5:
			//	b, err = sjson.SetBytes(b, "remark", data.Remark)
			//case 6,7:
			//	b, err = sjson.SetBytes(b, "detector_ip_source", data.IPSource.DetectorIPSource)
			//	b, err = sjson.SetBytes(b, "proxy_ip_list", data.IPSource.ProxyIPList)
			//	b, err = sjson.SetBytes(b, "proxy_ip_groups", data.IPSource.ProxyIPGroups)
			default:
			}
			if err != nil {
				return nil, fmt.Errorf("failed to sjson.SetBytes(%s): %s", b, err)
			}
		}

		if DEBUG {
			fmt.Printf("id:%d websitename:%s put payload => \n%s\n", id, data.Name, b)
		}

		b, err = cli.Update(b)
		if ok, e := api.OK2(b, err); !ok {
			return nil, fmt.Errorf("failed to update website: %s", e)
		}
	}
	return
}

func UpdateTransparentProxyWebsite(cli *tproxy.API, data *tproxy.ConfigOption, limit ...int) (b []byte, err error) {
	ids, e := cli.GetIdByName(data.Name)
	if e != nil {
		return nil, fmt.Errorf("failed to GetIdByName(%s): %s", data.Name, e)
	} else if len(ids) < 1 {
		return nil, fmt.Errorf("failed to GetIdByName(%s): %s", data.Name, "not found")
	}

	for _, id := range ids {
		obj, e := cli.GetDetailById(id)
		if ok, e := api.OK2(obj, e); !ok {
			return nil, fmt.Errorf("failed to GetDetailById(%d): %s", id, e)
		}

		result := gjson.GetBytes(obj, "data.0")
		b = []byte(result.Raw)
		for _, n := range limit {
			switch n {
			case 0:
				b, err = sjson.SetBytes(b, "is_enabled", data.IsEnabled)
			case 1:
				b, err = sjson.SetBytes(b, "policy_group", data.PolicyGroup)
			case 2:
				b, err = sjson.SetBytes(b, "detector_ip_source", data.IPSource.DetectorIPSource)
				b, err = sjson.SetBytes(b, "proxy_ip_list", data.IPSource.ProxyIPList)
				b, err = sjson.SetBytes(b, "proxy_ip_groups", data.IPSource.ProxyIPGroups)
			case 3:
				if data.Addrs[0].Ssl {
					b, err = sjson.SetBytes(b, "ssl_cert", data.SslCert)
				}
			case 4:
				b, err = sjson.SetBytes(b, "interface", data.Interface)
			case 5:
				b, err = sjson.SetBytes(b, "asset_group", data.AssetGroup)
			case 10:
				b, err = sjson.SetBytes(b, "access_log", data.AccessLog)
			//case 0:
			//	b, err = sjson.SetBytes(b, "is_enabled", data.IsEnabled)
			//case 1:
			//	b, err = sjson.SetBytes(b, "name", data.Name)
			////case 2,6,7:
			////	b, err = sjson.SetBytes(b, "addrs", data.Addrs)
			//case 3:
			//	b, err = sjson.SetBytes(b, "server_names", data.ServerNames)
			//case 4:
			//	b, err = sjson.SetBytes(b, "policy_group", data.PolicyGroup)
			//case 5:
			//	b, err = sjson.SetBytes(b, "interface", data.Interface)
			//case 8:
			//	b, err = sjson.SetBytes(b, "ssl_cert", data.SslCert)
			//case 9:
			//	b, err = sjson.SetBytes(b, "remark", data.Remark)
			//case 10,11:
			//	b, err = sjson.SetBytes(b, "detector_ip_source", data.IPSource.DetectorIPSource)
			//	b, err = sjson.SetBytes(b, "proxy_ip_list", data.IPSource.ProxyIPList)
			//	b, err = sjson.SetBytes(b, "proxy_ip_groups", data.IPSource.ProxyIPGroups)
			default:
			}
			if err != nil {
				return nil, fmt.Errorf("failed to sjson.SetBytes(%s): %s", b, err)
			}
		}

		if DEBUG {
			fmt.Printf("id:%d websitename:%s put payload => \n%s\n", id, data.Name, b)
		}

		b, err = cli.Update(b)
		if ok, e := api.OK2(b, err); !ok {
			return nil, fmt.Errorf("failed to update website: %s", e)
		}
	}
	return
}

func UpdateHardwareReverseProxyWebsite(cli *hproxy.API, data *hproxy.ConfigOption, limit ...int) (b []byte, err error) {
	ids, e := cli.GetIdByName(data.Name)
	if e != nil {
		return nil, fmt.Errorf("failed to GetIdByName(%s): %s", data.Name, e)
	} else if len(ids) < 1 {
		return nil, fmt.Errorf("failed to GetIdByName(%s): %s", data.Name, "not found")
	}

	for _, id := range ids {
		obj, e := cli.GetDetailById(id)
		if ok, e := api.OK2(obj, e); !ok {
			return nil, fmt.Errorf("failed to GetDetailById(%d): %s", id, e)
		}

		result := gjson.GetBytes(obj, "data.0")
		b = []byte(result.Raw)
		for _, n := range limit {
			switch n {
			case 0:
				b, err = sjson.SetBytes(b, "is_enabled", data.IsEnabled)
			case 1:
				b, err = sjson.SetBytes(b, "policy_group", data.PolicyGroup)
			case 2:
				b, err = sjson.SetBytes(b, "detector_ip_source", data.IPSource.DetectorIPSource)
				b, err = sjson.SetBytes(b, "proxy_ip_list", data.IPSource.ProxyIPList)
				b, err = sjson.SetBytes(b, "proxy_ip_groups", data.IPSource.ProxyIPGroups)
			case 3:
				if data.Ports[0].Ssl {
					b, err = sjson.SetBytes(b, "ssl_cert", data.SslCert)
				}
			case 4:
				b, err = sjson.SetBytes(b, "interface", data.Interface)
				b, err = sjson.SetBytes(b, "ip", data.IP)
			case 5:
				b, err = sjson.SetBytes(b, "asset_group", data.AssetGroup)

			case 6:
				b, err = sjson.SetBytes(b, "url_paths", data.URLPaths)
			case 7:
				b, err = sjson.SetBytes(b, "selected_tengine", data.SelectedTengine)
			case 8:
				b, err = sjson.SetBytes(b, "session_method", data.SessionMethod)
			case 9:
				b, err = sjson.SetBytes(b, "proxy_bind_config", data.ProxyBindConfig)
			case 10:
				b, err = sjson.SetBytes(b, "access_log", data.AccessLog)
			case 11:
				b, err = sjson.SetBytes(b, "backend_config.header_config", data.BackendConfig.HeaderConfig)
			case 12:
				b, err = sjson.SetBytes(b, "backend_config.keepalive_config", data.BackendConfig.KeepaliveConfig)
			case 13:
				b, err = sjson.SetBytes(b, "backend_config.x_forwarded_for_action", data.BackendConfig.XForwardedForAction)
			//case 0:
			//	b, err = sjson.SetBytes(b, "is_enabled", data.IsEnabled)
			//case 1:
			//	b, err = sjson.SetBytes(b, "name", data.Name)
			//case 2:
			//	b, err = sjson.SetBytes(b, "server_names", data.ServerNames)
			////case 3, 4, 5, 6:
			////	b, err = sjson.SetBytes(b, "ports", data.Ports)
			//case 7:
			//	b, err = sjson.SetBytes(b, "policy_group", data.SslCert)
			////case 8:
			////	b, err = sjson.SetBytes(b, "backend_config", data.BackendConfig)
			//case 9, 10:
			//	b, err = sjson.SetBytes(b, "interface", data.Interface)
			//	b, err = sjson.SetBytes(b, "ip", data.IP)
			//case 11:
			//	b, err = sjson.SetBytes(b, "policy_group", data.PolicyGroup)
			//case 12:
			//	b, err = sjson.SetBytes(b, "remark", data.Remark)
			//case 13,14:
			//	b, err = sjson.SetBytes(b, "detector_ip_source", data.IPSource.DetectorIPSource)
			//	b, err = sjson.SetBytes(b, "proxy_ip_list", data.IPSource.ProxyIPList)
			//	b, err = sjson.SetBytes(b, "proxy_ip_groups", data.IPSource.ProxyIPGroups)
			default:
			}
			if err != nil {
				return nil, fmt.Errorf("failed to sjson.SetBytes(%s): %s", b, err)
			}
		}

		if DEBUG {
			fmt.Printf("id:%d websitename:%s put payload => \n%s\n", id, data.Name, b)
		}

		b, err = cli.Update(b)
		if ok, e := api.OK2(b, err); !ok {
			return nil, fmt.Errorf("failed to update website: %s", e)
		}
	}
	return
}

func UpdateWebsite(a *api.API, filename string, mode string, limit ...int) {
	lengthCh := make(chan int)
	indexCh := make(chan int)
	stopCh := make(chan struct{})
	go progressBar(lengthCh, indexCh, stopCh)

	switch mode {
	case "TransparentBridge":
		cli := tbridge.NewFromAPI(a)
		rows := GenTransparentBridgeData(filename)
		for idx, line := range rows {
			_, err := UpdateTransparentBridgeWebsite(cli, line, limit...)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[error] row:%d websitename:%s %+v\n", idx+3, line.Name, err)
			}
			lengthCh <- len(rows)
			indexCh <- idx
		}
	case "TransparentProxy":
		cli := tproxy.NewFromAPI(a)
		rows := parseTransparentProxyExtraCsv(filename)
		for idx, line := range rows {
			_, err := UpdateTransparentProxyWebsite(cli, line, limit...)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[error] row:%d websitename:%s %+v\n", idx+3, line.Name, err)
			}
			lengthCh <- len(rows)
			indexCh <- idx
		}
	case "HardwareReverseProxy":
		cli := hproxy.NewFromAPI(a)
		rows := parseHardwareReverseProxyExtraCsv(filename)
		for idx, line := range rows {
			_, err := UpdateHardwareReverseProxyWebsite(cli, line, limit...)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[error] row:%d websitename:%s %+v\n", idx+3, line.Name, err)
			}
			lengthCh <- len(rows)
			indexCh <- idx
		}
	case "SoftwareReverseProxy":
		cli := hproxy.NewFromAPI(a)
		cli.URI = "/api/SoftwareReverseProxyWebsiteAPI"
		rows := parseHardwareReverseProxyExtraCsv(filename)
		for idx, line := range rows {
			//line.IP = []string{}
			line.Interface = ""
			_, err := UpdateHardwareReverseProxyWebsite(cli, line, limit...)
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
