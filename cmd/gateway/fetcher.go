package main

import (
	"fmt"
	"net/http"

	"github.com/clusterit/orca/cmd"
	"github.com/clusterit/orca/etcd"
	"github.com/clusterit/orca/users"

	"github.com/jmcvetta/napping"
)

type UserFetcher interface {
	UserByKey(zone, key string) (*users.User, error)
}

type httpFetcher struct {
	managerConfig etcd.Configurator
}

func NewHttpFetcher(cc *etcd.Cluster) (UserFetcher, error) {
	cfg, err := cc.NewManager()
	if err != nil {
		return nil, err
	}
	return &httpFetcher{managerConfig: cfg}, nil
}

func (hf *httpFetcher) UserByKey(zone, key string) (*users.User, error) {
	var u users.User
	urls, err := hf.managerConfig.GetValues("/" + cmd.ManagerService)
	if err != nil {
		return nil, err
	}
	if len(urls) < 1 {
		return nil, fmt.Errorf("no managers registered in configuration")
	}
	for _, url := range urls {
		r := napping.Request{
			Url:     fmt.Sprintf("%s/users/%s/pubkey", url, zone),
			Method:  "POST",
			Payload: key,
			Result:  &u,
			Header:  &http.Header{},
		}
		r.Header.Add("Content-Type", "application/json")
		r.Header.Add("Accept", "application/json")
		resp, err := napping.Send(&r)
		if err != nil {
			continue
		}
		if resp.Status()/100 == 5 {
			// the rest call returned an unknown error
			continue
		}
		if resp.Status() != 200 {
			return nil, fmt.Errorf("HTTP %d: %s", resp.Status(), resp.RawText())
		}
		return &u, nil
	}
	return nil, fmt.Errorf("no working manager found in configuration")
}
