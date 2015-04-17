![Orca](doc/img/orca_sm.png)

# Orca!
Hi! `orca`is a SSH gateway comparable to a reverse HTTP proxy. You 
can use `orca` to enable your employees to connect to internal servers without 
the need to build a VPN. Simply put `orca` in front of your servers and you can 
connect to any backend server via SSH.

![Call Graph](http://g.gravizo.com/g?
  digraph G {
   client -> sshgate [label="Agent Forwarding"];
   sshgate -> backend1;
   sshgate -> backend2;
   sshgate -> backend3 [label="Agent Forwarding"];
   sshgate [label="Orca Gateway"]
   backend1 [label="Backend 1"]
   backend2 [label="Backend 2"]
   backend3 [label="Backend 3"]
   sshgate -> keystore [style=dotted, label="HTTP Rest"];
   keystore [label="PubKey store"]
 }
)

`Orca` uses a public key store to query the user for a given public key. You can 
implement your own REST service (its one single function) or you can use the 
preimplemented key store bundled with `orca` which uses a clustered `etcd` 
backbone to store all the public keys.

## Usage

It is really simple to use. First of all you must have a SSH key pair and your 
client must have a running `ssh-agent` with the private key loaded. Second, your 
public key must be installed in the `authorized_keys` of your targeted backend 
servers. If you don't implement your own key store, you have to upload your 
public key to the `orca` keystore. This can be done with `orcaman` a simple 
webtool which uses OpenID Connect (OAuth2 for Login) to authenticate the user. 
If this is successful and the user was already authorized for the use of `orca` 
he can now upload his public keys.

Now the last part:
```sh
  ssh -A user@backend1@sshgw
```
and voilà: You're logged in to the Backend 1 server with your `user` account!

## Components
----------
`orca` has different components. Not all of them are needed, but it is simpler to use them at first.

### SSH Gateway
This is the SSH daemon that listens for incoming requests. Any request must have one or more 
public keys and the gateway invokes a REST call `UserByKey` to get the user for the given public
key. If there is no such user, the login will be denied.

If there is a user, the gateway checks an optinal *Allowance*. This check can be enforced, so that
the user has to login via a second channel (a web UI) and click on a button to allow the login for
a specific duration. 

After the permission to login to the gateway is granted, the request will be forwarded to the
referenced backend server with current `ssh-agent` provided keys. 

### OrcaMan
The `orcaman` provides Rest-Services and an embedded HTML5 app. This application can be used 
to store keys inside `etcd` and to request an*Allowance* for a specific time. If you have 
the **MANAGER** role you can also register new users and change some other settings. 
The `orcaman` also implements the `UserByKey` REST endpoint
which is needed by the gateway. If you start `orcaman` with a *publish address* it registeres
itself within a wellknown key inside of `etcd` so it will be discoverable by the gateway. After
stopping or crashing `orcaman` it will deregister automatically.

To secure the REST api, `orcaman` uses a *JWT* which signs the user data with a RSA key. The
allowed users for `orcaman` have to be configured in the `etcd` backbone. The user logs in via
an Oauth2 Provider (at this time `orcaman` supports *Google* users, but other OAuth2 providers will
follow soon) and `orcaman` checks if the user is allowed to use `orca`.

`orcaman` is also a cli webservice provider. It populates all the webservices with a 
simple *HTTP Basic Auth* backend. To disable other's to connect to this simple authentication
you should always put a reverse proxy in front of `orcaman`. You can also start the cli
services on a different interfaces (`--clilisten ...` so that they can only be used from localhost
whereas the Oauth-secured endpoints can listen on a public IP.

### Cli
The cli support some basic commands to manage the system from the command line.