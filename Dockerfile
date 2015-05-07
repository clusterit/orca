from fedora:latest
run yum -y install curl git mercurial make nodejs tar npm
run curl -L https://github.com/coreos/etcd/releases/download/v2.0.10/etcd-v2.0.10-linux-amd64.tar.gz | tar xz -C /
run curl https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz | tar xzC /usr/local
env PATH=/usr/local/go/bin:/etcd-v2.0.10-linux-amd64:$PATH
run npm install bower -g
run useradd orca
run mkdir /work
run mkdir /data
run chown orca:orca /work
run chown orca:orca /data
run git config --global url."https://".insteadOf git://
run cd /work && mkdir src pkg bin
env GOPATH=/work
env PATH=/work/bin:$PATH
run mkdir -p /work/src/github.com/clusterit/orca
add . /work/src/github.com/clusterit/orca/
run echo '{ "allow_root": true }' > /root/.bowerrc
run cd /work/src/github.com/clusterit/orca && make depends && make
expose 9011 22
volume /data
add scripts/test.sh /startup.sh
cmd /startup.sh
