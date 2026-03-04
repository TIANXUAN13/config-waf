package acl

import (
	"fmt"
	"net/url"

	"safeline/api"
)

type API struct {
	api.API
}

func New(baseUrl, token string) *API {
	return &API{
		*api.New(baseUrl, token, "/api/ACLRuleTemplateAPI"),
	}
}

type Data struct {
	Name         string `json:"name"`
	ExpirePeriod int    `json:"expire_period"`
	MatchMethod  struct {
		TargetType string      `json:"target_type"`
		Period     interface{} `json:"period"`
		Limit      interface{} `json:"limit"`
		Scope      string      `json:"scope"`
		Policy     string      `json:"policy"`
		Scheme     string      `json:"scheme"`
	} `json:"match_method"`
	Action struct {
		Action string `json:"action"`
	} `json:"action"`
	TemplateType string   `json:"template_type"`
	Targets      []string `json:"targets"`
}

func (api *API) CreateRule(data *Data) ([]byte, error) {
	return api.Post(data)
}

func (api *API) DeleteRule(id ...int) ([]byte, error) {
	data := struct {
		IDIn []int `json:"id__in"`
	}{IDIn: id}
	return api.Delete(data)
}

func (api API) AddIp(id int, targets ...string) ([]byte, error) {
	api.URI = "/api/ACLRuleAPI"
	data := struct {
		AclRuleTemplateId int           `json:"acl_rule_template_id"`
		Targets           []string      `json:"targets"`
		TargetIpGroups    []interface{} `json:"target_ip_groups"`
	}{id, targets, make([]interface{}, 0)}
	return api.Post(data)
}

func (api API) GetIp(id int) ([]byte, error) {
	api.URI = "/api/ACLRuleAPI"
	q := url.Values{}
	q.Add("acl_rule_template_id", fmt.Sprintf("%d", id))
	return api.Get(q)
}

func (api API) DelIp(id int) ([]byte, error) {
	api.URI = "/api/ACLRuleAPI"
	data := struct {
		Id             int  `json:"id"`
		AddToWhiteList bool `json:"add_to_white_list"`
	}{id, false}
	return api.Delete(data)
}

func (api API) DelAllIp(id int) ([]byte, error) {
	api.URI = "/api/ClearACLRuleAPI"
	data := struct {
		AclRuleTemplateId int  `json:"acl_rule_template_id"`
		AddToWhiteList    bool `json:"add_to_white_list"`
	}{AclRuleTemplateId: id, AddToWhiteList: false}
	return api.Delete(data)
}
