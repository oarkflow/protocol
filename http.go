package protocol

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/oarkflow/errors"

	"github.com/oarkflow/protocol/http"
)

type HTTP struct {
	client  *http.Client
	Config  *http.Options
	Service string
}

func (s *HTTP) Setup() error {
	timeout := 1 * time.Second
	if s.Config.URL == "" {
		return errors.New("empty url")
	}
	parse, err := url.Parse(s.Config.URL)
	if err != nil {
		return err
	}
	port := "80"
	if parse.Port() != "" {
		port = parse.Port()
	}
	_, err = net.DialTimeout("tcp", parse.Hostname()+":"+port, timeout)
	if err != nil {
		return fmt.Errorf("site unreachable %s", err)
	}
	return nil
}

func (s *HTTP) GetType() Type {
	return Http
}

func (s *HTTP) SetService(service Service) {

}

func (s *HTTP) GetServiceType() string {
	return s.Service
}

func (s *HTTP) Queue(payload Payload) (Response, error) {
	return s.Handle(payload)
}

func (s *HTTP) Handle(payload Payload) (Response, error) {
	if payload.URL == "" {
		payload.URL = s.Config.URL
	}
	if payload.Method == "" {
		payload.Method = s.Config.Method
	}
	switch strings.ToUpper(payload.Method) {
	case "POST":
		response, err := s.client.Post(payload.URL, payload.Data, payload.Headers)
		if err != nil {
			return nil, err
		}
		bt, _ := io.ReadAll(response.Body)
		if response.StatusCode >= 400 && response.StatusCode < 600 {
			return nil, errors.New(string(bt))
		}
		return bt, nil
	case "PUT":
		response, err := s.client.Put(payload.URL, payload.Data, payload.Headers)
		if err != nil {
			return nil, err
		}
		bt, _ := io.ReadAll(response.Body)
		if response.StatusCode >= 400 && response.StatusCode < 600 {
			return nil, errors.New(string(bt))
		}
		return bt, nil
	case "HEAD":
		response, err := s.client.Head(payload.URL, payload.Headers)
		if err != nil {
			return nil, err
		}
		bt, _ := io.ReadAll(response.Body)
		if response.StatusCode >= 400 && response.StatusCode < 600 {
			return nil, errors.New(string(bt))
		}
		return bt, nil
	case "FORM":
		response, err := s.client.Form(payload.URL, payload.Data, payload.Headers)
		if err != nil {
			return nil, err
		}
		bt, _ := io.ReadAll(response.Body)
		if response.StatusCode >= 400 && response.StatusCode < 600 {
			return nil, errors.New(string(bt))
		}
		return bt, nil
	default:
		response, err := s.client.Get(payload.URL, payload.Data, payload.Headers)
		if err != nil {
			return nil, err
		}
		bt, _ := io.ReadAll(response.Body)
		if response.StatusCode >= 400 && response.StatusCode < 600 {
			return nil, errors.New(string(bt))
		}
		return bt, nil
	}
}
