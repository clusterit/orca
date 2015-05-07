package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/clusterit/orca/config"
	"github.com/clusterit/orca/logging"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type clientSession struct {
	serverConn        *ssh.ServerConn
	agent             agent.Agent
	remoteUser        string
	remoteHost        string
	remotePort        int
	backend           *backendClient
	bufferedRqs       []*ssh.Request
	globalBufferedRqs []*ssh.Request
	logger            *logging.Logger
}

type backendClient struct {
	orcasession *clientSession
	client      *ssh.Client
	session     *ssh.Session
	newchans    <-chan ssh.NewChannel
	requests    <-chan *ssh.Request
	mux         sync.Mutex
	handlers    map[string]chan ssh.NewChannel
}

func newClient(conn ssh.Conn, chans <-chan ssh.NewChannel, rqs <-chan *ssh.Request) *backendClient {
	newchans := make(map[string]chan ssh.NewChannel)
	cl := &backendClient{client: &ssh.Client{Conn: conn}, newchans: chans, requests: rqs, handlers: newchans}
	go cl.handleGlobalRequests(rqs)
	go cl.handleChannelOpens(chans)
	return cl
}

func (c *backendClient) handle(tp string) <-chan ssh.NewChannel {
	c.mux.Lock()
	defer c.mux.Unlock()

	ch := c.handlers[tp]
	if ch != nil {
		return ch
	}
	ch = make(chan ssh.NewChannel, 16)
	c.handlers[tp] = ch
	return ch
}

func (c *backendClient) handleGlobalRequests(incoming <-chan *ssh.Request) {
	for r := range incoming {
		c.orcasession.tracef("new inoming rq: %#v", r)
		r.Reply(false, nil)
	}
}

func (c *backendClient) handleChannelOpens(in <-chan ssh.NewChannel) {
	for ch := range in {
		c.orcasession.tracef("new inoming channel: %s", ch.ChannelType())
		c.mux.Lock()
		handler := c.handlers[ch.ChannelType()]
		c.mux.Unlock()

		if handler != nil {
			handler <- ch
		} else {
			ch.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %v", ch.ChannelType()))
		}
	}

	c.mux.Lock()
	for _, ch := range c.handlers {
		close(ch)
	}
	c.handlers = make(map[string]chan ssh.NewChannel)
	c.mux.Unlock()
}

func (c *backendClient) forwardAgent(ag agent.Agent) error {
	return agent.ForwardToAgent(c.client, ag)
}

func (c *backendClient) newSession() error {
	session, err := c.client.NewSession()
	if err != nil {
		return err
	}
	c.session = session
	return nil
}

func (c *backendClient) close() error {
	c.session.Close()
	return c.client.Close()
}

func NewSession(sshConn *ssh.ServerConn, chans <-chan ssh.NewChannel, reqs <-chan *ssh.Request, err error) (*clientSession, error) {
	if err != nil {
		return nil, err
	}

	var cs clientSession
	cs.serverConn = sshConn
	cs.remotePort = 22
	cs.remoteUser, cs.remoteHost, err = split(sshConn.User())
	if err != nil {
		sshConn.Close()
		return nil, err
	}
	err = checkBackendAccess(cs.remoteHost, *configuration)
	if err != nil {
		sshConn.Close()
		return nil, err
	}
	remote := sshConn.RemoteAddr().String()
	sid := fmt.Sprintf("%x", sshConn.SessionID())
	cs.logger = logging.New(sid, remote)
	cs.infof("new ssh connection with %s ", sshConn.ClientVersion())

	go func() {
		for rq := range reqs {
			cs.forward(rq, true)
		}
	}()
	//go ssh.DiscardRequests(reqs)
	go cs.handleChannels(sshConn, chans)

	return &cs, nil
}

func (cs *clientSession) handleChannels(con ssh.Conn, chans <-chan ssh.NewChannel) {
	for newChannel := range chans {
		if newChannel.ChannelType() == "session" {
			channel, requests, err := newChannel.Accept()
			if err != nil {
				cs.errorf("accepting new session channel: %s", err)
				con.Close()
				continue
			}
			go cs.handleChannel(con, channel, requests)
		} else if newChannel.ChannelType() == "direct-tcpip" {
			channel, requests, err := newChannel.Accept()
			if err != nil {
				cs.errorf("accepting new direct-tcpip channel: %s", err)
				con.Close()
				continue
			}
			go ssh.DiscardRequests(requests)
			cs.tunnelChannel(channel, newChannel.ChannelType(), newChannel.ExtraData())
		} else {
			cs.errorf("unknown channel: %#v", newChannel)
			newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", newChannel.ChannelType()))
		}
	}
}

func (cs *clientSession) handleChannel(con ssh.Conn, channel ssh.Channel, reqs <-chan *ssh.Request) {
	defer func() {
		e := recover()
		if e != nil {
			cs.errorf("handle Channel: %s (is there an agent running on client?)", e)
			con.Close()
		}
	}()
	for req := range reqs {
		cs.tracef("channel request: %s", req.Type)
		if strings.HasPrefix(req.Type, "auth-agent-req") {
			subs := string([]byte(req.Type)[len("auth-agent-req"):])
			rq := fmt.Sprintf("auth-agent%s", subs)
			ac, _, err := con.OpenChannel(rq, nil)
			if err != nil {
				panic(err)
			}
			cs.agent = agent.NewClient(ac)
			_, err = cs.connectToBackend(fmt.Sprintf("%s:%d", cs.remoteHost, cs.remotePort), cs.remoteUser, cs.agent)
			if err != nil {
				panic(err)
			}

			if req.WantReply {
				req.Reply(true, nil)
			}
		} else {
			switch req.Type {
			case "exec":
				exc := parseStrings(req.Payload, 1)
				cs.debugf("ssh exec: %v", exc)
				go cs.connectRemote(fmt.Sprintf("%s:%d", cs.remoteHost, cs.remotePort), channel, &exc[0])
			case "shell":
				go cs.connectRemote(fmt.Sprintf("%s:%d", cs.remoteHost, cs.remotePort), channel, nil)
			default:
				cs.forward(req, false)
				//log.Printf("[DEBUG] req: %+v", req)
			}
		}
	}
}

func (cs *clientSession) forward(rq *ssh.Request, global bool) error {
	if cs.backend == nil {
		if global {
			cs.globalBufferedRqs = append(cs.globalBufferedRqs, rq)
		} else {
			cs.bufferedRqs = append(cs.bufferedRqs, rq)
		}
		return nil
	} else {
		if global {
			return cs.forwardGlobalRequest(rq)
		}
		return cs.forwardRequest(rq)
	}
}

func (cs *clientSession) connectToBackend(backend string, user string, ag agent.Agent) (*backendClient, error) {
	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.PublicKeysCallback(ag.Signers)},
	}

	cs.debugf("connect to backend %s with user %s", backend, user)
	client, err := dial("tcp", backend, sshConfig)

	if err != nil {
		return nil, fmt.Errorf("Dial error: %s", err)
	}
	cs.backend = client
	client.orcasession = cs

	for _, rq := range cs.globalBufferedRqs {
		if err := cs.forwardGlobalRequest(rq); err != nil {
			cs.errorf("send global %s to backend: %s", rq.Type, err)
		}
	}
	cs.globalBufferedRqs = nil

	err = client.forwardAgent(ag)
	if err != nil {
		return nil, fmt.Errorf("agentforward error: %s", err)
	}
	cs.debugf("opening new clientsession for user %s", user)
	err = client.newSession()
	if err != nil {
		client.close()
		return nil, fmt.Errorf("session error: %s", err)
	}

	go cs.handleBackendChannel("x11", client.handle("x11"))
	go cs.handleBackendChannel("forwarded-tcpip", client.handle("forwarded-tcpip"))

	return client, nil
}

func (cs *clientSession) connectRemote(backend string, channel ssh.Channel, cmd *string) {
	defer cs.serverConn.Close()
	if cs.backend == nil {
		cs.errorf("you must enable agent forwarding")
		return
	}
	defer cs.backend.close()

	sess := cs.backend.session

	cs.forwardBufferedRequest()
	stdoutP, e := sess.StdoutPipe()
	if e != nil {
		cs.errorf("connect to stdout: %s", e)
		return
	}
	stderrP, e := sess.StderrPipe()
	if e != nil {
		cs.errorf("connect to stderr: %s", e)
		return
	}
	stdinP, e := sess.StdinPipe()
	if e != nil {
		cs.errorf("connect to stdin: %s", e)
		return
	}
	go io.Copy(channel, stdoutP)
	go io.Copy(channel.Stderr(), stderrP)
	go io.Copy(stdinP, channel)
	if cmd != nil {
		cs.debugf("start remote command %s", *cmd)
		e = sess.Start(*cmd)
	} else {
		cs.debugf("opening remote shell")
		e = sess.Shell()
	}
	if e != nil {
		cs.errorf("opening shell: %s", e)
		return
	}
	e = sess.Wait()
	exitCode := 0
	if e != nil {
		ee, ok := e.(*ssh.ExitError)
		if ok {
			wm := ee.Waitmsg
			cs.errorf("wait for shell, signal:%s, status:%d, msg:%s", wm.Signal(), wm.ExitStatus(), wm.Msg())
			exitCode = wm.ExitStatus()
		} else {
			cs.errorf("wait for shell: %s", e)
			exitCode = 255 // ???
		}
		return
	}
	if cmd != nil {
		// TODO: send signal back!
		resbuf := make([]byte, 4)
		binary.BigEndian.PutUint32(resbuf, uint32(exitCode))
		channel.SendRequest("exit-status", false, resbuf)
	}
}

func (cs *clientSession) handleBackendChannel(tp string, nch <-chan ssh.NewChannel) error {
	for ch := range nch {
		c, rqs, err := ch.Accept()
		cs.tracef("new backendchannel '%s'", tp)
		if err != nil {
			return fmt.Errorf("[ERROR] backendChannel Accept: %s", err)
		}
		go ssh.DiscardRequests(rqs)
		clientChannel, crqs, err := cs.serverConn.OpenChannel(ch.ChannelType(), ch.ExtraData())
		if err != nil {
			return fmt.Errorf("[ERROR] opening channel %s to client : %s", tp, err)
		}
		go func() {
			go ssh.DiscardRequests(crqs)
			go io.Copy(c, clientChannel)
			io.Copy(clientChannel, c)
			clientChannel.Close()
			c.Close()
		}()
	}
	return nil
}

func (cs *clientSession) forwardGlobalRequest(rq *ssh.Request) error {
	ok, data, err := cs.backend.client.SendRequest(rq.Type, rq.WantReply, rq.Payload)
	if err != nil {
		return err
	} else {
		cs.tracef("forwarded global %s to backend [%v]", rq.Type, ok)
		if rq.WantReply {
			rq.Reply(ok, data)
		}
	}
	return nil
}

func (cs *clientSession) forwardRequest(rq *ssh.Request) error {
	ok, err := cs.backend.session.SendRequest(rq.Type, rq.WantReply, rq.Payload)
	if err != nil {
		return err
	} else {
		cs.tracef("forwarded %s to backend [%v]", rq.Type, ok)
		if rq.WantReply {
			rq.Reply(ok, nil)
		}
	}
	return nil
}

func (cs *clientSession) forwardBufferedRequest() error {
	for _, rq := range cs.bufferedRqs {
		if err := cs.forwardRequest(rq); err != nil {
			cs.errorf("send %s to backend: %s", rq.Type, err)
		}
	}
	cs.bufferedRqs = make([]*ssh.Request, 0)
	return nil
}

func (cs *clientSession) tunnelChannel(ch ssh.Channel, tp string, data []byte) error {
	ch1, rqs, err := cs.backend.client.OpenChannel(tp, data)
	if err != nil {
		return fmt.Errorf("open channel %s to backend: %s", tp, err)
	}
	go ssh.DiscardRequests(rqs)
	go func() {
		go io.Copy(ch1, ch)
		io.Copy(ch, ch1)
		ch.Close()
		ch1.Close()
	}()
	return nil

}

func split(userAtHost string) (string, string, error) {
	res := strings.Split(userAtHost, "@")
	if len(res) != 2 {
		if configuration.DefaultHost != "" {
			return userAtHost, configuration.DefaultHost, nil
		}
		return "", "", fmt.Errorf("unknown target: %s", userAtHost)
	}
	return res[0], res[1], nil
}

func parseStrings(b []byte, num int) []string {
	var res []string
	if len(b) < 1 {
		return res
	}
	offset := 0
	for i := 0; i < num; i++ {
		len := int(binary.BigEndian.Uint32(b[offset:]))
		start := offset + 4
		end := start + len
		res = append(res, string(b[start:end]))
		offset = end
	}
	return res
}

func dial(network, addr string, config *ssh.ClientConfig) (*backendClient, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return nil, err
	}
	return newClient(c, chans, reqs), err
}

func checkBackendAccess(address string, cfg config.Gateway) error {
	ips, err := net.LookupIP(address)
	if err != nil {
		return err
	}
	var allowed []*net.IPNet
	var denied []*net.IPNet
	for _, a := range cfg.AllowedCidrs {
		_, netw, err := net.ParseCIDR(a)
		if err == nil {
			allowed = append(allowed, netw)
		} else {
			logger.Warnf("the allowed CIDR %s cannot be parsed, ignoring", a)
		}
	}
	for _, a := range cfg.DeniedCidrs {
		_, netw, err := net.ParseCIDR(a)
		if err == nil {
			denied = append(denied, netw)
		} else {
			logger.Warnf("the denied CIDR %s cannot be parsed, ignoring", a)
		}
	}

	a := checkNetContains(allowed, ips)
	d := checkNetContains(denied, ips)

	if cfg.AllowDeny {
		// if it is allowed
		if a {
			// and not denied
			if !d {
				// allow it
				return nil
			}
		}

		return fmt.Errorf("AD: %s is not allowed: allowd:%v, denied:%v", address, a, d)
	}

	if d {
		// if it is denied, deny it
		if a {
			// except it is allowed
			return nil
		}
		return fmt.Errorf("DA: %s is not allowed: allowd:%v, denied:%v", address, a, d)
	}
	return nil
}

func checkNetContains(nets []*net.IPNet, ips []net.IP) bool {
	for _, nw := range nets {
		for _, ip := range ips {
			if nw.Contains(ip) {
				return true
			}
		}
	}
	return false
}
