package main

import (
	"encoding/json"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"

	"safeline/api"
	"safeline/api/rule"
)

type ruleResp struct {
	Err  interface{} `json:"err"`
	Data []rule.Data `json:"data"`
	Msg  interface{} `json:"msg"`
}

type ruleItems struct {
	Items []rule.Data `json:"items"`
	Total int         `json:"total"`
}

func getRule(client *api.API) ([]byte, error) {
	client.URI = "/api/PolicyRuleAPI"
	return client.Get(nil)
}

func DisplayGlobalRule(cli *api.API, w *os.File, raw bool, args ...interface{}) {
	displayRule(cli, w, raw, true)
}

func DisplayCustomRule(cli *api.API, w *os.File, raw bool, args ...interface{}) {
	displayRule(cli, w, raw, false)
}

func displayRule(cli *api.API, w *os.File, raw bool, global bool) {
	b, err := getRule(cli)
	panicIf(err)
	if ok, err := api.OK(b); !ok {
		panic(err)
	}
	resp := &ruleResp{}
	err = json.Unmarshal(b, resp)
	panicIf(err)

	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.AppendHeader(table.Row{"序号", "开启", "规则ID", "动作", "备注", "过期时间", "创建时间"})
	data := ruleItems{}
	for idx, v := range resp.Data {
		if (global && !v.IsGlobal) || (!global && v.IsGlobal) {
			// 过滤全局规则或者过滤自定义规则
			// 雷池全局和自定义规则是同一个接口
			// 通过 v.IsGlobal 来确定是否是全局还是自定义
			continue
		}
		if raw {
			data.Total += 1
			data.Items = append(data.Items, v)
			continue
		}
		expire := "永不过期"
		if v.ExpireTime != nil {
			expire = formatTime(*v.ExpireTime)
		}
		t.AppendRow([]interface{}{idx + 1, v.IsEnabled, v.ID, v.Action, v.Comment, expire, formatTime(v.CreateTime)})
	}
	if raw {
		r, _ := json.MarshalIndent(data, "", "\t")
		_, _ = w.Write(r)
		return
	}
	if w.Name() != os.Stdout.Name() {
		t.RenderCSV()
	} else {
		t.Render()
	}
}
