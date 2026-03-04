package main

import (
	"encoding/json"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"

	"safeline/api"
)

type ipGroupResp struct {
	Err  interface{} `json:"err"`
	Data []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"data"`
	Msg interface{} `json:"msg"`
}

func getIpGroup(client *api.API) ([]byte, error) {
	client.URI = "/api/IPGroupAPI"
	return client.Get(nil)
}

func DisplayIpGroup(cli *api.API, w *os.File, raw bool, args ...interface{}) {
	b, err := getIpGroup(cli)
	if ok, err := api.OK2(b, err); !ok {
		panic(err)
	}
	if raw {
		_, _ = w.Write(b)
		return
	}

	group := &ipGroupResp{}
	panicIf(json.Unmarshal(b, group))

	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.AppendHeader(table.Row{"序号", "IP组ID", "IP组名称"})

	for idx, g := range group.Data {
		t.AppendRow([]interface{}{idx + 1, g.ID, g.Name})
	}

	if w.Name() != os.Stdout.Name() {
		t.RenderCSV()
	} else {
		t.Render()
	}
}
