package main

import (
	"encoding/json"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"

	"safeline/api"
)

type aclRuleResp struct {
	Err  interface{} `json:"err"`
	Data []struct {
		Id        int    `json:"id"`
		Name      string `json:"name"`
		IsEnabled bool   `json:"is_enabled"`
		Action    struct {
			Action          string `json:"action"`
			LimitRateLimit  int    `json:"limit_rate_limit,omitempty"`
			LimitRatePeriod int    `json:"limit_rate_period,omitempty"`
		} `json:"action"`
		CreateTime      string `json:"create_time"`
		ExpirePeriod    *int   `json:"expire_period"`
		ExecutionNumber int    `json:"execution_number"`
		TemplateType    string `json:"template_type"`
		IsInaccurate    bool   `json:"is_inaccurate"`
		DryRun          bool   `json:"dry_run"`
	} `json:"data"`
	Msg interface{} `json:"msg"`
}

func getAclRule(client *api.API) ([]byte, error) {
	client.URI = "/api/ACLRuleTemplateAPI"
	return client.Get(nil)
}

func DisplayAclRule(cli *api.API, w *os.File, raw bool, args ...interface{}) {
	b, err := getAclRule(cli)
	if ok, err := api.OK2(b, err); !ok {
		panic(err)
	}
	if raw {
		_, _ = w.Write(b)
		return
	}

	resp := &aclRuleResp{}
	panicIf(json.Unmarshal(b, resp))

	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.AppendHeader(table.Row{"序号", "状态", "ID", "名称", "封禁方式", "限制时间(s)", "创建时间"})

	for idx, v := range resp.Data {
		var expire int
		if v.ExpirePeriod != nil {
			expire = *v.ExpirePeriod
		}
		t.AppendRow([]interface{}{idx + 1, v.IsEnabled, v.Id, v.Name, v.Action.Action, expire, formatTime(v.CreateTime)})
	}
	if w.Name() != os.Stdout.Name() {
		t.RenderCSV()
	} else {
		t.Render()
	}
}
