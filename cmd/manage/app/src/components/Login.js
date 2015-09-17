import React from 'react';
import UserActions from '../actions/Users';
import UserStore from '../stores/Users';
import mui from 'material-ui';

let RaisedButton = mui.RaisedButton;

export default class Login extends React.Component {
  constructor(props) {
		super(props);
    this.handleLogin = ::this.handleLogin;
  }

  componentDidMount () {
    this.unsubscribe = UserStore.listen(this.onLogin);
  }
  componentWillUnmount () {
    this.unsubscribe();
  }

  onLogin () {
    console.log("handle login");
  }

  handleLogin(e) {
    UserActions.login();
  }

  render() {
    return (
      <div>
        <h2>Login</h2>
        <RaisedButton label="Login" primary={true} onTouchTap={this.handleLogin} />
      </div>
    );
  }
}
