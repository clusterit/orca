package testsupport

import (
	"os"

	"github.com/clusterit/orca/etcd"
)

import "github.com/samalba/dockerclient"

const (
	docksock = "unix:///var/run/docker.sock"
)

type TS interface {
	StartEtcd() (*etcd.Cluster, error)
	StopEtcd() error
}

type ts struct {
	client *dockerclient.DockerClient
	etcdid string
}

func New() (TS, error) {
	docker, e := dockerclient.NewDockerClient(docksock, nil)
	if e != nil {
		return nil, e
	}
	return &ts{client: docker}, nil
}

func (t *ts) StartEtcd() (*etcd.Cluster, error) {
	etcdServer := os.Getenv("TEST_ETCD_MACHINE")
	if etcdServer == "" {
		if t.etcdid != "" {
			t.StopEtcd()
		}
		containerConfig := &dockerclient.ContainerConfig{
			Image: "elcolio/etcd:latest",
		}
		containerId, err := t.client.CreateContainer(containerConfig, "")
		if err != nil {
			return nil, err
		}
		t.etcdid = containerId
		hostConfig := &dockerclient.HostConfig{}

		err = t.client.StartContainer(containerId, hostConfig)
		if err != nil {
			t.StopEtcd()
			return nil, err
		}
		info, err := t.client.InspectContainer(containerId)
		if err != nil {
			t.StopEtcd()
			return nil, err
		}
		etcdServer = "http://" + info.NetworkSettings.IPAddress + ":4001"
	}
	cls, err := etcd.Init([]string{etcdServer})
	if err != nil {
		t.StopEtcd()
		return nil, err
	}
	return cls, nil
}

func (t *ts) StopEtcd() error {
	t.client.StopContainer(t.etcdid, 0)
	t.client.KillContainer(t.etcdid, "SIGKILL")
	t.client.RemoveContainer(t.etcdid, true, true)
	return nil
}
