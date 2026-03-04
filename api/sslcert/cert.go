package sslcert

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"path/filepath"

	"safeline/api"
)

type API struct {
	api.API
}

func New(baseUrl, token string) *API {
	return &API{
		*api.New(baseUrl, token, "/api/CertAPI"),
	}
}

func (cli *API) Fetch() ([]byte, error) {
	return cli.Get(nil)
}

func (cli *API) Remove(id ...int) ([]byte, error) {
	data := struct {
		IDIn []int `json:"id__in"`
	}{IDIn: id}
	return cli.Delete(data)
}

func (cli API) Upload(filenames []string, password string) ([]byte, error) {
	body := bytes.NewBuffer(nil)
	w := multipart.NewWriter(body)
	var part io.Writer
	var err error
	for idx, name := range filenames {
		if idx == 0 {
			part, err = w.CreateFormFile("crt_file", filepath.Base(name))
		} else if idx == 1 {
			part, err = w.CreateFormFile("key_file", filepath.Base(name))
		}
		if err != nil {
			return nil, err
		}
		b, err := ioutil.ReadFile(name)
		if err != nil {
			return nil, err
		}
		_, err = part.Write(b)
		if err != nil {
			return nil, err
		}
	}
	if password != "" {
		err = w.WriteField("password", password)
		_, err = part.Write([]byte(password))
	}
	if err = w.Close(); err != nil {
		return nil, err
	}
	cli.URI = "/api/UploadSSLCertAPI"
	cli.Header.Set("Content-Type", w.FormDataContentType())
	return cli.Do("POST", body.Bytes())
}
