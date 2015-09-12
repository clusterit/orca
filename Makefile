DEPENDENCIES = \
	github.com/coreos/go-etcd/etcd \
  github.com/spf13/cobra \
	github.com/Sirupsen/logrus \
	gopkg.in/errgo.v1 \
	github.com/satori/go.uuid \
	github.com/emicklei/go-restful \
	github.com/ulrichSchreiner/authkit \
	github.com/GeertJohan/go.rice

deps:
	go get $(DEPENDENCIES)

depsupdate:
	go get -u $(DEPENDENCIES)
