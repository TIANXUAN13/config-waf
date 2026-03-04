package api

import (
	"encoding/json"
	"errors"
)

type Response struct {
	Err  interface{} `json:"err"`
	Data interface{} `json:"data"`
	Msg  interface{} `json:"msg"`
}

func OK(data []byte) (bool, error) {
	r := &Response{}
	e := json.Unmarshal(data, r)
	if e != nil {
		return false, e
	}
	if _, ok := r.Err.(string); ok {
		return false, errors.New(string(data))
	}
	return true, nil
}

func OK2(data []byte, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	return OK(data)
}
