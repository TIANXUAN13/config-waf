package website

import (
	"encoding/json"
	"net/url"
	"strconv"
	"sync"

	"github.com/tidwall/gjson"

	"safeline/api"
)

type IPSource struct {
	DetectorIPSource []string
	ProxyIPList      []string
	ProxyIPGroups    []int
}

type API struct {
	api.API
	siteNameMap map[string][]int
	once        sync.Once
	err         error
}

func (cli *API) GetIdByName(name ...string) ([]int, error) {
	ret := make([]int, 0)
	cli.once.Do(cli.getId)
	if cli.err != nil {
		return ret, cli.err
	}
	for _, k := range name {
		if v, ok := cli.siteNameMap[k]; ok {
			ret = append(ret, v...)
		}
	}
	return ret, nil
}

func (cli *API) getId() {
	cli.siteNameMap = make(map[string][]int, 0)
	b, err := cli.Get(nil)
	if ok, err := api.OK2(b, err); !ok {
		cli.err = err
		return
	}
	rsp := struct {
		Err  interface{} `json:"err"`
		Data []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
		Msg interface{} `json:"msg"`
	}{}

	if err = json.Unmarshal(b, &rsp); err != nil {
		cli.err = err
		return
	}

	for _, data := range rsp.Data {
		k := data.Name
		v := data.ID
		if _, ok := cli.siteNameMap[k]; ok {
			cli.siteNameMap[k] = append(cli.siteNameMap[k], v)
		} else {
			cli.siteNameMap[k] = []int{v}
		}
	}
}

func (cli *API) GetIdByNameLike(name ...string) ([]int, error) {
	ret := make([]int, 0)
	q := url.Values{}
	for _, v := range name {
		q.Add("name__like", v)
	}
	b, err := cli.Get(q)
	if ok, err := api.OK2(b, err); !ok {
		return ret, err
	}
	ids := gjson.GetBytes(b, "data.#.id").Array()
	for _, id := range ids {
		ret = append(ret, int(id.Int()))
	}
	return ret, nil
}

func (cli *API) GetDetailById(id ...int) ([]byte, error) {
	q := url.Values{}
	for _, v := range id {
		q.Add("id__in", strconv.Itoa(v))
	}
	return cli.Get(q)
}

func (cli *API) Update(rawData []byte) ([]byte, error) {
	return cli.Do("PUT", rawData)
}

func (cli *API) Remove(id ...int) ([]byte, error) {
	data := struct {
		IDIn []int `json:"id__in"`
	}{IDIn: id}
	return cli.Delete(data)
}
