"use strict";

import React from 'react';
import mui from 'material-ui';
import { RouteHandler } from 'react-router';
import Login from './components/Login';
import UserStore from './stores/Users';

let ThemeManager = new mui.Styles.ThemeManager();
let AppBar = mui.AppBar
  , LeftNav = mui.LeftNav
  , MenuItem = mui.MenuItem
  , Avatar = mui.Avatar
  , RaisedButton = mui.RaisedButton;

let menuItems = [
    { route: '/', text: 'Account' },
    { route: 'about', text: 'About' },
    { route: 'contact', text: 'Contact' },
];

export default class OrcaApp extends React.Component {
	constructor(props) {
		super(props);
    this.handleClick = ::this.handleClick;
    this.getSelectedIndex = ::this.getSelectedIndex;
    this.onLeftNavChange = ::this.onLeftNavChange;
    this.state = {
      user : null
    };
	}

  componentDidMount() {
    this.unsubscribe = UserStore.listen(::this.onLogin);
  }
  componentWillUnmount () {
    this.unsubscribe();
  }
  onLogin (u) {
    this.setState ({user:u});
    console.log("handle login:",u);
  }

  getChildContext() {
    return {
      muiTheme: ThemeManager.getCurrentTheme()
    };
  }

  handleClick(e) {
    e.preventDefault();
    this.refs.leftNav.toggle();
  }

  getSelectedIndex() {
    let currentItem;

    for (let i = menuItems.length - 1; i >= 0; i--) {
      currentItem = menuItems[i];
      if (currentItem.route && this.context.router.isActive(currentItem.route)) {
        return i;
      }
    }
  }

  onLeftNavChange(e, key, payload) {
    // Do DOM Diff refresh
    this.context.router.transitionTo(payload.route);
  }

	render() {
    if (!this.state.user) {
      return (
        <Login></Login>
      )
    }
    else {
      var avatarStyles = {marginLeft:'10px', marginTop:'10px'};
      let header = (
        <Avatar style={avatarStyles} src={this.state.user.thumbnail} />
      );
  		return (
        <div id="page_container">
          <LeftNav
              ref="leftNav"
              docked={false}
              menuItems={menuItems}
              selectedIndex={this.getSelectedIndex()}
              onChange={this.onLeftNavChange}
              header={header} />
          <header>
             <AppBar title='Orca' onLeftIconButtonTouchTap={this.handleClick} />
          </header>
          <section className="content">
            <RouteHandler />
          </section>
        </div>
  		);
    }
	}
}

OrcaApp.childContextTypes = {
  muiTheme: React.PropTypes.object
};

OrcaApp.contextTypes = {
  router: React.PropTypes.func
};
