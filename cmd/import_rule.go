package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"safeline/api"
	"safeline/api/rule"
)

func loadJson(filename string) (*ruleItems, error) {
	rd, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer rd.Close()

	b, err := ioutil.ReadAll(rd)
	if err != nil {
		return nil, err
	}
	it := &ruleItems{}
	err = json.Unmarshal(b, it)
	if err != nil {
		return nil, err
	}
	return it, nil
}

func ImportRule(a *api.API, filename string, global bool) {
	cli := rule.NewFromAPI(a)
	data, err := loadJson(filename)
	panicIf(err)
	success := 0
	fail := 0
	for _, v := range data.Items {
		v.IsGlobal = global
		if ok, err := api.OK2(cli.Create(&v)); !ok {
			fail++
			fmt.Printf("导入规则:\t'%s' ... 导入失败!\n => %s\n", v.Comment, err)
			continue
		}
		success++
		fmt.Printf("导入规则:\t'%s' ... 导入成功!\n", v.Comment)
	}
	fmt.Printf("共导入 %d 条规则，成功: %d条, 失败: %d条.\n", success+fail, success, fail)
}
