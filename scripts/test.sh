#!/bin/sh

cd /data
etcd -name etcd1 -listen-client-urls http://localhost:4001 -advertise-client-urls http://localhost:4001 -listen-peer-urls http://localhost:7001 -initial-advertise-peer-urls http://localhost:7001 -initial-cluster-token etcd-cluster-1 -initial-cluster 'etcd1=http://localhost:7001' -initial-cluster-state new &

# give etcd some time to initialize
sleep 3

if [ ! -d "/data/etcd1.etcd" ]; then
  /work/src/github.com/clusterit/orca/packaging/orcaman provider github $CLIENTID $CLIENTSECRET
  /work/src/github.com/clusterit/orca/packaging/orcaman admins github:$USERID 
fi

/work/src/github.com/clusterit/orca/packaging/orcaman serve >orcaman.logs 2>&1 &
export ORCA_BIND=0.0.0.0:22
/work/src/github.com/clusterit/orca/packaging/sshgw serve >sshgw.logs 2>&1

