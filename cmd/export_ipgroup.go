package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	urlpkg "net/url"
	"os"
	"strings"

	"safeline/api"
)

func ExportIpGroupIP(cli *api.API, filename, format string) error {
	items, err := getAllIpGroupDetails(cli)
	if err != nil {
		return err
	}

	u, _ := urlpkg.Parse(cli.BaseUrl)
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "csv"
	}

	switch format {
	case "json":
		name := fmt.Sprintf("%s_%s.json", filename, u.Hostname())
		f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		return enc.Encode(items)
	case "csv":
		name := fmt.Sprintf("%s_%s.csv", filename, u.Hostname())
		f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		_, _ = f.WriteString("\xEF\xBB\xBF")
		w := csv.NewWriter(f)
		w.UseCRLF = true
		defer w.Flush()

		if err := w.Write([]string{"IP组ID", "IP组名称", "备注", "CIDR", "原始IP"}); err != nil {
			return err
		}
		for _, item := range items {
			cLen, oLen := len(item.Cidrs), len(item.Original)
			rowCount := cLen
			if oLen > rowCount {
				rowCount = oLen
			}
			if rowCount == 0 {
				if err := w.Write([]string{intToString(item.ID), item.Name, item.Comment, "", ""}); err != nil {
					return err
				}
				continue
			}
			for i := 0; i < rowCount; i++ {
				cidr, original := "", ""
				if i < cLen {
					cidr = item.Cidrs[i]
				}
				if i < oLen {
					original = item.Original[i]
				}
				if err := w.Write([]string{intToString(item.ID), item.Name, item.Comment, cidr, original}); err != nil {
					return err
				}
			}
		}
		return w.Error()
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}
