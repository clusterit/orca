import React from 'react';
import UserActions from '../actions/Users';
import mui from 'material-ui';

let RaisedButton = mui.RaisedButton;

var pageStyle = {
  display: 'flex',
  flexDirection : 'column',
  justifyContent : 'center',
  background: 'white',
  height: '100%'
};

var rowStyle = {
  display : 'flex',
  flexDirection : 'row',
  justifyContent : 'center'
}

var supportedLogins = {
  google: 'Google'
};

export default class Login extends React.Component {
  constructor(props) {
		super(props);
    this.handleLogin = ::this.handleLogin;
    var provs = authkit.providers();
    this.providers = Object.keys(provs).map(p => this.provider(provs[p]));
  }

  provider (p) {
    var lbl = supportedLogins[p.network];
    if (lbl) {
      p.label = lbl;
    } else {
      p.label = p.network;
    }
    return p;
  }

  handleLogin(p) {
    UserActions.login(p);
  }

  render() {
    return (
      <div style={pageStyle}>
        <h3 style={rowStyle}>Login with</h3>
        <div>&nbsp;</div>
        <div style={rowStyle}>
        {this.providers.map(p =>
          <RaisedButton key={p.network} label={p.label} primary={false} onTouchTap={this.handleLogin.bind(this,p)} />
        )}
        </div>
      </div>
    );
  }
}
