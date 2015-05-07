# Quickstart

For a simple testdrive of orca you can use docker and pull the image
`ulrichschreiner/orca-testdrive`. Orca uses an external OAuth provider 
and supports `github`, so it is the simplest, you 
[register a new application](https://github.com/settings/applications/new). Enter
*Orca local* as name and *http://localhost:9011* as homepage URL. The redirect
url should be *http://localhost:9011/redirect.html*. Now click on `Update Application`
and you will get a `Client ID` and a `Client Secret`. As the name says, keep the
secret in a secret place.

Now pull the image with `docker pull ulrichschreiner/orca-testdrive` and
run the image. The following line assumes you have the ClientID and ClientSecret
of your registered application in the environmentvariables `GITHUB_CLIENTID` and
`GITHUB_CLIENTSECRET`. You also need your github userid in `GITHUB_USERID`. The
included `etcd` stores its values in your `$HOME` folder in the `data` directory:

```
docker run -p 9011:9011 -p 2022:22 -v $HOME/data:/data -e CLIENTID=$GITHUB_CLIENTID -e CLIENTSECRET=$GITHUB_CLIENTSECRET -e USERID=$GITHUB_USERID ulrichschreiner/orca-testdrive
```

Now point your browser to *http://localhost:9011* and login!