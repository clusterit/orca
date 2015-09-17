import Reflux from 'reflux';
import UserActions from '../actions/Users';

var UserStore = Reflux.createStore({
  init: function() {
    this.listenTo(UserActions.login, 'onLogin');
  },
  onLogin : function () {
    console.log("UserStore.onLogin");
    this.trigger("login trigger");
  }
});

module.exports = UserStore;
