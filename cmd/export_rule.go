package main

import (
	"fmt"
	"net/url"
	"os"

	"safeline/api"
)

func ExportRule(a *api.API, filename string, global bool) {
	u, _ := url.Parse(a.BaseUrl)
	name := fmt.Sprintf("%s_%s.json", filename, u.Hostname())
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY, 0644)
	panicIf(err)
	defer f.Close()
	displayRule(a, f, true, global)
}
