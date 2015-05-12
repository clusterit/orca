package etcd

import (
	"fmt"
	"testing"
	"time"

	"github.com/clusterit/orca/testsupport"
	"github.com/clusterit/orca/users"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	pubkey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDyLg8zuWzJOgcTru78NkhhDsa+tjasjrJGoJbhBRMHrxgwdgUF5ZKGsV2LWTZgp8rIDUjHRGWSlvTrpXCG33wmRJrXwxYG3J0QeOAYRlMD3ESBVtPWm2iqA02PzpL7+mnmV79Ml3Q8yUz8Ef5Bs+lytVAw42IhfTEfJyWM9zsjFEW/NvZ6cttrOUhwEQ1r9HvY0UDyHRA3sW0B3I2KfYg1Z1e5wlKDd7dGI9u/S9E9JwFpeh/AXjPiN/Vd2xInIh99G9HsWBdpTaNlYXZj6Qnx/wLcCm2v7U9WdIvM5M+xqiYZ6pxGUtsBDgBjraxh8tRWV3eab3stZsKnwQthyp4P title"
)

func TestUser(t *testing.T) {
	ts, e := testsupport.New()
	if e != nil {
		t.Fatalf("cannot init etcd container: %s", e)
	}
	cluster, e := ts.StartEtcd()
	if e != nil {
		t.Fatalf("cannot start etcd container: %s", e)
	}
	defer ts.StopEtcd()
	userimpl, e := New(cluster)
	if e != nil {
		t.Fatalf("cannot create etcd cluster: %s", e)
	}

	Convey("Create a few users", t, func() {
		usr, err := userimpl.Create("network", "id", "name", users.ManagerRoles)
		So(err, ShouldBeNil)
		So(usr.Name, ShouldEqual, "name")
		So(usr.Aliases[0], ShouldEqual, "id@network")
		So(usr.Name, ShouldEqual, "name")
		So(usr.Roles, ShouldContain, users.RoleManager)
		usr, err = userimpl.Update("id@network", "newname", users.UserRoles)
		So(err, ShouldBeNil)
		So(usr.Name, ShouldEqual, "newname")
		So(usr.Roles, ShouldNotContain, users.RoleManager)
		usr, err = userimpl.Create("mynetwork", "myid", "myname", users.UserRoles)
		So(err, ShouldBeNil)
		So(usr.Name, ShouldEqual, "myname")
		So(usr.Aliases[0], ShouldEqual, "myid@mynetwork")
		Convey("and read all users", func() {
			usrs, err := userimpl.GetAll()
			So(err, ShouldBeNil)
			So(len(usrs), ShouldEqual, 2)
			Convey("now delete one user", func() {
				_, err := userimpl.Delete("id@network")
				So(err, ShouldBeNil)
				usrs, err := userimpl.GetAll()
				So(err, ShouldBeNil)
				So(len(usrs), ShouldEqual, 1)
				So(usrs[0].Id, ShouldEqual, usr.Id)
			})
			Convey("and permit the user to login", func() {
				t := time.Now().Add(time.Hour)
				a := users.Allowance{GrantedBy: "kruemelmonster", Uid: "myid@mynetwork", Until: t}
				err := userimpl.Permit(a, 3600)
				So(err, ShouldBeNil)
				u, err := userimpl.Get(usr.Id)
				So(err, ShouldBeNil)
				So(u.Allowance, ShouldNotBeNil)
				So(u.Allowance.Until, ShouldHappenOnOrBetween, t, t.Add(time.Second))
			})
		})
		Convey("play with aliases", func() {
			usr, err := userimpl.AddAlias("myid@mynetwork", "myothernetwork", "myid2")
			So(err, ShouldBeNil)
			So(usr.Aliases, ShouldContain, "myid@mynetwork")
			So(usr.Aliases, ShouldContain, "myid2@myothernetwork")
			usr, err = userimpl.RemoveAlias("myid@mynetwork", "myothernetwork", "myid2")
			So(err, ShouldBeNil)
			So(usr.Aliases, ShouldContain, "myid@mynetwork")
			So(usr.Aliases, ShouldNotContain, "myid2@myothernetwork")
		})
		Convey("check if the idtoken returns the same user", func() {
			usr2, err := userimpl.ByIdToken(usr.IdToken)
			So(err, ShouldBeNil)
			So(usr2.Id, ShouldEqual, usr.Id)
			Convey("and also if we create a new token, the user should be the same", func() {
				usr2, err = userimpl.NewIdToken(usr.Id)
				So(err, ShouldBeNil)
				So(usr.IdToken, ShouldNotEqual, usr2.IdToken)
				usr3, err := userimpl.ByIdToken(usr2.IdToken)
				So(err, ShouldBeNil)
				So(usr3.Id, ShouldEqual, usr.Id)
			})
		})
		Convey("enable 2FA and tokencheck", func() {
			uimpl := userimpl.(*etcdUsers)
			scratchToken := 12121212
			uimpl.scratchCodes = []int{scratchToken}
			url, err := userimpl.Create2FAToken("mydomain", "myid@mynetwork")
			So(err, ShouldBeNil)
			So(url, ShouldContainSubstring, "myname")
			So(url, ShouldContainSubstring, "mydomain")
			// the next check only tests if there is a secreot for the user
			err = userimpl.CheckToken("myid@mynetwork", fmt.Sprintf("%d", scratchToken))
			So(err, ShouldBeNil)
			err = userimpl.Use2FAToken("myid@mynetwork", true)
			So(err, ShouldBeNil)
			u, err := userimpl.SetAutologinAfter2FA("myid@mynetwork", 10)
			So(err, ShouldBeNil)
			So(u.Use2FA, ShouldBeTrue)
			So(u.AutologinAfter2FA, ShouldEqual, 10)
			n := time.Now()
			err = userimpl.CheckAndAllowToken("myid@mynetwork", fmt.Sprintf("%d", scratchToken), 100)
			So(err, ShouldBeNil)
			u, err = userimpl.Get("myid@mynetwork")
			So(err, ShouldBeNil)
			So(u.Allowance, ShouldNotBeNil)
			So(u.Allowance.Until, ShouldHappenBetween, n, n.Add(12*time.Second))
		})
		Convey("add two keys", func() {
			pk, err := users.ParseKey(pubkey)
			So(err, ShouldBeNil)
			k, err := userimpl.AddKey("myid@mynetwork", "kid", pk.Value, pk.Fingerprint)
			So(err, ShouldBeNil)
			So(k.Id, ShouldEqual, "kid")
			k2, err := userimpl.AddKey(usr.Id, "kid2", "keycontent2", "key-fingerprin2t")
			So(err, ShouldBeNil)
			So(k2.Id, ShouldEqual, "kid2")
			Convey("now read the user and check if there are both keys", func() {
				u, err := userimpl.Get(usr.Id)
				So(err, ShouldBeNil)
				So(len(u.Keys), ShouldEqual, 2)
				var keys []string
				for _, k := range u.Keys {
					keys = append(keys, k.Id)
				}
				So(keys, ShouldContain, "kid")
				So(keys, ShouldContain, "kid2")
				Convey("try to find user by key", func() {
					u, uk, err := userimpl.GetByKey(pubkey)
					So(err, ShouldBeNil)
					So(u.Id, ShouldEqual, usr.Id)
					So(uk.Id, ShouldEqual, k.Id)
				})
				Convey("removing one key should succeed too", func() {
					_, err := userimpl.RemoveKey("myid@mynetwork", k.Id)
					So(err, ShouldBeNil)
					u, err := userimpl.Get(usr.Id)
					So(err, ShouldBeNil)
					So(len(u.Keys), ShouldEqual, 1)
					var keys []string
					for _, k := range u.Keys {
						keys = append(keys, k.Id)
					}
					So(keys, ShouldNotContain, "kid")
					So(keys, ShouldContain, "kid2")
					Convey("use helper function to add key", func() {
						_, e := users.AsKey(userimpl, "myid@mynetwork", "", pubkey)
						So(e, ShouldBeNil)
						u, err := userimpl.Get(usr.Id)
						So(err, ShouldBeNil)
						So(len(u.Keys), ShouldEqual, 2)
						var keys []string
						for _, k := range u.Keys {
							keys = append(keys, k.Id)
						}
						So(keys, ShouldContain, "title")
						So(keys, ShouldContain, "kid2")
					})
				})
			})
		})
	})
}
