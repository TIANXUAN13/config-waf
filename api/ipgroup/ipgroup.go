package ipgroup

import (
	"net/url"
	"strconv"

	"safeline/api"
)

type API struct {
	api.API
}

func New(baseUrl, token string) *API {
	return &API{
		*api.New(baseUrl, token, "/api/IPGroupAPI"),
	}
}

type Data struct {
	Name     string   `json:"name"`
	Comment  string   `json:"comment"`
	Original []string `json:"original"`
	Id       int      `json:"id,omitempty"`
}

func (cli *API) Create(data *Data) ([]byte, error) {
	return cli.Post(data)
}

func (cli *API) Update(data *Data) ([]byte, error) {
	return cli.Put(data)
}

func (cli *API) Remove(id ...int) ([]byte, error) {
	data := struct {
		IDIn []int `json:"id__in"`
	}{IDIn: id}
	return cli.Delete(data)
}

func (cli API) AddIp(id int, targets ...string) ([]byte, error) {
	cli.URI = "/api/EditIPGroupItem"
	data := struct {
		Id      int      `json:"id"`
		Targets []string `json:"targets"`
	}{id, targets}
	return cli.Post(data)
}

func (cli API) DeleteIp(id int, targets ...string) ([]byte, error) {
	cli.URI = "/api/EditIPGroupItem"
	data := struct {
		Id      int      `json:"id"`
		Targets []string `json:"targets"`
	}{id, targets}
	return cli.API.Delete(data)
}

func (cli API) ListDetail(count, offset int, scopes ...string) ([]byte, error) {
	cli.URI = "/api/FilterV2API"
	scope := "detect:asset:ip_group"
	if len(scopes) > 0 && scopes[0] != "" {
		scope = scopes[0]
	}
	query := url.Values{}
	query.Set("count", strconv.Itoa(count))
	query.Set("offset", strconv.Itoa(offset))
	query.Set("scope", scope)
	return cli.Get(query)
}
