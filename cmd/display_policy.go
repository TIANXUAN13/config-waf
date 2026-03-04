package main

import (
	"encoding/json"
	"github.com/jedib0t/go-pretty/v6/table"
	"os"

	"safeline/api"
)

type policyResp struct {
	Err  interface{} `json:"err"`
	Data []struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		CreateTime string `json:"create_time"`
	}
	Msg interface{} `json:"msg"`
}

func getPolicyGroup(client *api.API) ([]byte, error) {
	client.URI = "/api/PolicyGroupAPI"
	return client.Get(nil)
}

func DisplayPolicyGroup(cli *api.API, w *os.File, raw bool, args ...interface{}) {
	b, err := getPolicyGroup(cli)
	if ok, err := api.OK2(b, err); !ok {
		panic(err)
	}
	if raw {
		_, _ = w.Write(b)
		return
	}

	group := &policyResp{}
	panicIf(json.Unmarshal(b, group))

	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.AppendHeader(table.Row{"序号", "策略ID", "策略名", "创建时间"})

	for idx, g := range group.Data {
		t.AppendRow([]interface{}{idx + 1, g.ID, g.Name, formatTime(g.CreateTime)})
	}

	if w.Name() != os.Stdout.Name() {
		t.RenderCSV()
	} else {
		t.Render()
	}
}
