package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"

	"safeline/api"
)

type certResp struct {
	Err  interface{} `json:"err"`
	Data []struct {
		ID         int           `json:"id"`
		Websites   []interface{} `json:"websites"`
		CreateTime string        `json:"create_time"`
		Name       string        `json:"name"`
	} `json:"data"`
	Msg interface{} `json:"msg"`
}

func getSSLCert(client *api.API) ([]byte, error) {
	client.URI = "/api/CertAPI"
	return client.Get(nil)
}

func DisplaySSLCert(cli *api.API, w *os.File, raw bool, args ...interface{}) {
	b, err := getSSLCert(cli)
	if ok, err := api.OK2(b, err); !ok {
		panic(err)
	}
	if raw {
		_, _ = w.Write(b)
		return
	}

	resp := &certResp{}
	panicIf(json.Unmarshal(b, resp))

	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.AppendHeader(table.Row{"序号", "证书ID", "证书名称", "创建时间", "正在使用的站点"})

	for idx, v := range resp.Data {
		if len(v.Websites) > 0 {
			var ret []string
			for _, value := range v.Websites {
				if r, ok := value.(map[string]interface{}); ok {
					ret = append(ret, fmt.Sprintf("%s(id:%.f)", r["name"], r["id"]))
				}
			}
			t.AppendRow([]interface{}{idx + 1, v.ID, v.Name, formatTime(v.CreateTime), strings.Join(ret, "\n")})
		} else {
			t.AppendRow([]interface{}{idx + 1, v.ID, v.Name, formatTime(v.CreateTime), "无站点使用"})
		}
	}
	if w.Name() != os.Stdout.Name() {
		t.RenderCSV()
	} else {
		t.Render()
	}
}
