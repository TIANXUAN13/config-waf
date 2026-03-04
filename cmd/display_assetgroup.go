package main

import (
	"encoding/json"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"

	"safeline/api"
)

type assetGroupResp struct {
	Err  interface{} `json:"err"`
	Data []struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		Comment    string `json:"comment"`
		Priority   int    `json:"priority"`
		CreateTime string `json:"create_time"`
	} `json:"data"`
	Msg interface{} `json:"msg"`
}

func getAssetGroupResp(client *api.API) ([]byte, error) {
	client.URI = "/api/waf_assets/v1/Group"
	return client.Get(nil)
}

func DisplayAssetGroup(cli *api.API, w *os.File, raw bool, args ...interface{}) {
	b, err := getAssetGroupResp(cli)
	if ok, err := api.OK2(b, err); !ok {
		panic(err)
	}
	if raw {
		_, _ = w.Write(b)
		return
	}

	group := &assetGroupResp{}
	panicIf(json.Unmarshal(b, group))

	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.AppendHeader(table.Row{"序号", "web资产组ID", "web资产组名称", "备注"})
	for idx, g := range group.Data {
		t.AppendRow([]interface{}{idx + 1, g.ID, g.Name, g.Comment})
	}

	if w.Name() != os.Stdout.Name() {
		t.RenderCSV()
	} else {
		t.Render()
	}
}
