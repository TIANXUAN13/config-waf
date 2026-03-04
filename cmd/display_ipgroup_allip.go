package main

import (
	"encoding/json"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"

	"safeline/api"
)

func DisplayIpGroupAllIp(cli *api.API, w *os.File, raw bool) error {
	items, err := getAllIpGroupDetails(cli)
	if err != nil {
		return err
	}
	if raw {
		b, err := json.Marshal(items)
		if err != nil {
			return err
		}
		_, _ = w.Write(b)
		return nil
	}

	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.AppendHeader(table.Row{"序号", "IP组ID", "IP组名称", "备注", "IP数量", "IP列表"})

	for idx, item := range items {
		ips := item.Cidrs
		if len(ips) == 0 {
			ips = item.Original
		}
		t.AppendRow(table.Row{idx + 1, item.ID, item.Name, item.Comment, len(ips), listToString(ips)})
	}

	if w.Name() != os.Stdout.Name() {
		t.RenderCSV()
	} else {
		t.Render()
	}
	return nil
}
