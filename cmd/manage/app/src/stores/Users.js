import Reflux from 'reflux';
import UserActions from '../actions/Users';

var UserStore = Reflux.createStore({
  currentUser : null,

  init: function() {
    this.listenTo(UserActions.login, 'onLogin');
  },
  onLogin : function (p) {
    console.log("UserStore.onLogin:",p);
    var self = this;
    authkit.login(p.network).user(function (usr, tok) {
      console.log("logged in:",usr);
      self.currentUser = usr;
      self.trigger(usr);
    });
  }
});

module.exports = UserStore;
