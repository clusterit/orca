import Reflux from 'reflux';
import UserActions from '../actions/Users';

var request = require('superagent');

var UserStore = Reflux.createStore({
  currentUser : null,
  currentNetwork : null,

  init: function() {
    this.listenTo(UserActions.login, 'onLogin');
  },
  onLogin : function (p) {
    console.log("UserStore.onLogin:",p);
    var self = this;
    authkit.login(p.network).user(function (usr, tok) {
      console.log("logged in:",usr);
      self.currentUser = usr;
      self.currentNetwork = p.network;
      request.
        get("/api/v1/users/"+p.network+"/"+usr.email).
        set('Accept', 'application/json').
        end(function(err, res) {
          if (!err)
            console.log("find user:",res.body)
        });
      self.trigger(usr);
    });
  },
});

module.exports = UserStore;
