"use strict";

import React from 'react';
import mui from 'material-ui';
import { RouteHandler } from 'react-router';

let ThemeManager = new mui.Styles.ThemeManager();
let AppBar = mui.AppBar
  , LeftNav = mui.LeftNav
  , MenuItem = mui.MenuItem
  , RaisedButton = mui.RaisedButton;

let menuItems = [
    { route: '/', text: 'Home' },
    { route: 'about', text: 'About' },
    { route: 'contact', text: 'Contact' },
];

export default class OrcaApp extends React.Component {
	constructor(props) {
		super(props);
    this.handleClick = ::this.handleClick;
    this.getSelectedIndex = ::this.getSelectedIndex;
    this.onLeftNavChange = ::this.onLeftNavChange;
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
		return (
      <div id="page_container">
        <LeftNav
            ref="leftNav"
            docked={false}
            menuItems={menuItems}
            selectedIndex={this.getSelectedIndex()}
            onChange={this.onLeftNavChange} />
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

OrcaApp.childContextTypes = {
  muiTheme: React.PropTypes.object
};

OrcaApp.contextTypes = {
  router: React.PropTypes.func
};
