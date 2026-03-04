package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"safeline/api"
	"safeline/api/acl"
	"safeline/api/ipgroup"
)

func AddIpByGroupId(cli *api.API, groupId int, targets ...string) ([]byte, error) {
	ipGroupCli := ipgroup.New(cli.BaseUrl, cli.Token)
	return ipGroupCli.AddIp(groupId, targets...)
}

func AddIpByGroupName(cli *api.API, name string, targets ...string) ([]byte, error) {
	b, err := getIpGroup(cli)
	if ok, err := api.OK2(b, err); !ok {
		return b, err
	}
	group := &ipGroupResp{}
	err = json.Unmarshal(b, group)
	if err != nil {
		return b, err
	}
	var id int
	for _, g := range group.Data {
		if g.Name == name {
			id = g.ID
			break
		}
	}
	ipGroupCli := ipgroup.New(cli.BaseUrl, cli.Token)
	return ipGroupCli.AddIp(id, targets...)
}

func AddIpByAclRuleId(cli *api.API, aclId int, targets ...string) ([]byte, error) {
	aclCli := acl.New(cli.BaseUrl, cli.Token)
	return aclCli.AddIp(aclId, targets...)
}

func AddIpByAclRuleName(cli *api.API, name string, targets ...string) ([]byte, error) {
	b, err := getAclRule(cli)
	if ok, err := api.OK2(b, err); !ok {
		return b, err
	}
	rule := &aclRuleResp{}
	err = json.Unmarshal(b, rule)
	if err != nil {
		return b, err
	}

	var id int
	for _, r := range rule.Data {
		if r.Name == name {
			id = r.Id
			break
		}
	}
	aclCli := acl.New(cli.BaseUrl, cli.Token)
	return aclCli.AddIp(id, targets...)
}

func DelAllIpByAclRuleId(cli *api.API, aclId int) ([]byte, error) {
	aclCli := acl.New(cli.BaseUrl, cli.Token)
	return aclCli.DelAllIp(aclId)
}

type aclIpResp struct {
	Err  interface{} `json:"err"`
	Data []struct {
		Id              int         `json:"id"` // item id
		TargetIpGroup   interface{} `json:"target_ip_group"`
		AclRuleTemplate struct {
			Id           int         `json:"id"` // rule id
			Name         string      `json:"name"`
			Version      int         `json:"version"`
			IsEnabled    bool        `json:"is_enabled"`
			CreateTime   string      `json:"create_time"`
			ExpirePeriod interface{} `json:"expire_period"`
		} `json:"acl_rule_template"`
		Target     string      `json:"target"`
		ExpireTime interface{} `json:"expire_time"`
		CreateTime string      `json:"create_time"`
	} `json:"data"`
	Msg interface{} `json:"msg"`
}

type ipGroupDetailResp struct {
	Err  interface{} `json:"err"`
	Data struct {
		Items []ipGroupDetailItem `json:"items"`
		Total int                 `json:"total"`
	} `json:"data"`
	Msg interface{} `json:"msg"`
}

type ipGroupDetailItem struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Comment  string   `json:"comment"`
	Cidrs    []string `json:"cidrs"`
	Original []string `json:"original"`
}

func getAllIpGroupDetails(cli *api.API) ([]ipGroupDetailItem, error) {
	ipGroupCli := ipgroup.New(cli.BaseUrl, cli.Token)
	count := 200
	out := make([]ipGroupDetailItem, 0, count)
	for offset := 0; ; offset += count {
		b, err := ipGroupCli.ListDetail(count, offset)
		if ok, err := api.OK2(b, err); !ok {
			return nil, err
		}
		resp := &ipGroupDetailResp{}
		if err := json.Unmarshal(b, resp); err != nil {
			return nil, err
		}
		out = append(out, resp.Data.Items...)
		if offset+len(resp.Data.Items) >= resp.Data.Total || len(resp.Data.Items) == 0 {
			break
		}
	}
	return out, nil
}

func getIpId(cli *acl.API, aclId int, targets ...string) []int {
	b, err := cli.GetIp(aclId)
	if ok, err := api.OK2(b, err); !ok {
		panic(err)
	}
	resp := &aclIpResp{}
	panicIf(json.Unmarshal(b, resp))

	var ret []int
	for _, r := range resp.Data {
		for _, v := range targets {
			if r.Target == v {
				ret = append(ret, r.Id)
				break
			}
		}
	}
	return ret
}

func DelIpByAclRuleId(cli *api.API, aclId int, targets ...string) ([]byte, error) {
	aclCli := acl.New(cli.BaseUrl, cli.Token)
	ids := getIpId(aclCli, aclId, targets...)
	for _, id := range ids {
		b, err := aclCli.DelIp(id)
		if ok, err := api.OK2(b, err); !ok {
			fmt.Fprintf(os.Stderr, "[error] %v\n", err.Error())
		}
	}
	return nil, nil
}

func DelIpByIpGroupId(cli *api.API, groupId int, targets ...string) ([]byte, error) {
	ipGroupCli := ipgroup.New(cli.BaseUrl, cli.Token)
	b, err := ipGroupCli.DeleteIp(groupId, targets...)
	if ok, err := api.OK2(b, err); !ok {
		return b, err
	}
	return nil, nil
}

func DisplayIpByAclRuleId(cli *api.API, id int) {
	aclCli := acl.New(cli.BaseUrl, cli.Token)
	b, err := aclCli.GetIp(id)
	if ok, err := api.OK2(b, err); !ok {
		panic(err)
	}
	resp := &aclIpResp{}
	err = json.Unmarshal(b, resp)
	if err != nil {
		panic(err)
	}
	for _, r := range resp.Data {
		fmt.Println(r.Target)
	}
}

func DisplayIpByIpGroupId(cli *api.API, id int) error {
	items, err := getAllIpGroupDetails(cli)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.ID != id {
			continue
		}
		if len(item.Cidrs) > 0 {
			for _, v := range item.Cidrs {
				fmt.Println(v)
			}
			return nil
		}
		for _, v := range item.Original {
			fmt.Println(v)
		}
		return nil
	}
	return errors.New("ipgroup not found")
}

func CreateAclRule(cli *api.API, name string, expire int, targets ...string) ([]byte, error) {
	aclCli := acl.New(cli.BaseUrl, cli.Token)
	data := &acl.Data{Name: name, ExpirePeriod: expire, Targets: targets}
	data.TemplateType = "manual"
	data.Action.Action = "forbid"
	data.MatchMethod.Limit = 1
	data.MatchMethod.Period = 5
	data.MatchMethod.Policy = ""
	data.MatchMethod.Scheme = "http(s)"
	data.MatchMethod.TargetType = "CIDR"
	data.MatchMethod.Scope = "All"
	b, err := aclCli.CreateRule(data)
	if ok, err := api.OK2(b, err); !ok {
		return b, err
	}
	return nil, nil
}

func DeleteAclRule(cli *api.API, id ...int) ([]byte, error) {
	aclCli := acl.New(cli.BaseUrl, cli.Token)
	b, err := aclCli.DeleteRule(id...)
	if ok, err := api.OK2(b, err); !ok {
		return b, err
	}
	return nil, nil
}

func CreateIpGroup(cli *api.API, name, comment string, targets ...string) ([]byte, error) {
	ipGroupCli := ipgroup.New(cli.BaseUrl, cli.Token)
	data := &ipgroup.Data{
		Name:     name,
		Comment:  comment,
		Original: targets,
	}
	b, err := ipGroupCli.Create(data)
	if ok, err := api.OK2(b, err); !ok {
		return b, err
	}
	return nil, nil
}

func DeleteIpGroup(cli *api.API, id ...int) ([]byte, error) {
	ipGroupCli := ipgroup.New(cli.BaseUrl, cli.Token)
	b, err := ipGroupCli.Remove(id...)
	if ok, err := api.OK2(b, err); !ok {
		return b, err
	}
	return nil, nil
}
