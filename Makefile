SHA := $(shell git rev-parse HEAD)
OUTROOT := packaging

all: build

complete: build embed

build: gateway webman climan orcacli embed

gateway:
	go build -o $(OUTROOT)/sshgw -ldflags "-X main.revision $(SHA)" github.com/clusterit/orca/cmd/gateway

webman:
	go build -o $(OUTROOT)/webman -ldflags "-X main.revision $(SHA)" github.com/clusterit/orca/cmd/webman
	
climan:
	go build -o $(OUTROOT)/climan -ldflags "-X main.revision $(SHA)" github.com/clusterit/orca/cmd/climan

orcacli:
	go build -o $(OUTROOT)/orcacli -ldflags "-X main.revision $(SHA)" github.com/clusterit/orca/cmd/cli

embed:
	rice --import-path="github.com/clusterit/orca/cmd/webman" append --exec="$(OUTROOT)/webman"


depends:
	go get github.com/GeertJohan/go.rice/rice
	
clean:
	rm -rf $(OUTROOT)/*