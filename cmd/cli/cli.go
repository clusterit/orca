package main

import (
	"fmt"
	"io/ioutil"

	"github.com/clusterit/orca/config"
	"github.com/clusterit/orca/users"

	"github.com/jmcvetta/napping"
)

func (c *cli) unmarshal(rq *napping.Request, target interface{}) error {
	resp, err := c.session.Send(rq)
	if err != nil {
		return err
	}
	if resp.Status() != 200 {
		return fmt.Errorf("HTTP %d: %s", resp.Status(), resp.RawText())
	}
	if target != nil {
		return resp.Unmarshal(target)
	}
	return nil
}

func (c *cli) createUser(id, name string, roles ...string) error {
	rlz := make([]users.Role, len(roles))
	for i, r := range roles {
		rlz[i] = users.Role(r)
	}
	t := users.User{Id: id, Name: name, Roles: rlz}
	r := c.rq("PUT", "/users", t)
	return c.unmarshal(r, nil)
}

func (c *cli) listUsers() ([]users.User, error) {
	var res []users.User
	r := c.rq("GET", "/users", nil)
	return res, c.unmarshal(r, &res)
}

func (c *cli) parseKey(k string) (*users.Key, error) {
	var key users.Key
	r := c.rq("POST", "/users/parsekey", k)
	return &key, c.unmarshal(r, &key)
}

func (c *cli) addKey(uid, keyname string, file string) error {
	kf, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	if keyname == "" {
		k, err := c.parseKey(string(kf))
		if err != nil {
			return err
		}
		keyname = k.Id
	}
	// zone hardcoded, cause orca don't uses zones for users
	r := c.rq("PUT", fmt.Sprintf("/users/%s/%s/zone/pubkey", uid, keyname), string(kf))
	return c.unmarshal(r, nil)
}

func (c *cli) deleteKey(uid, keyname string) error {
	// zone hardcoded, cause orca don't uses zones for users
	r := c.rq("DELETE", fmt.Sprintf("/users/%s/%s/zone/pubkey", uid, keyname), nil)
	return c.unmarshal(r, nil)
}

func (c *cli) zones() ([]string, error) {
	var res []string
	r := c.rq("GET", "/configuration/zones", nil)
	return res, c.unmarshal(r, &res)
}

func (c *cli) getGateway(stage string) (*config.Gateway, error) {
	var res config.Gateway
	r := c.rq("GET", fmt.Sprintf("/configuration/%s/gateway", stage), nil)
	return &res, c.unmarshal(r, &res)
}

func (c *cli) putGateway(stage string, gw config.Gateway) error {
	r := c.rq("PUT", fmt.Sprintf("/configuration/%s/gateway", stage), gw)
	return c.unmarshal(r, nil)
}
