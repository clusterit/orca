"use strict";
import React from 'react';
import mui from 'material-ui';
import injectTapEventPlugin from 'react-tap-event-plugin';
import Router, { Route, DefaultRoute, NotFoundRoute, Redirect, Link } from 'react-router';


injectTapEventPlugin();

// Our application
import OrcaApp from './OrcaApp';
import Account from './components/Account';
import About from './components/About';
import Contact from './components/Contact';

const AppRoutes = (
  <Route path="/" handler={OrcaApp}>
    <DefaultRoute handler={Account} />
    <Route name="about" handler={About} />
    <Route name="contact" handler={Contact} />
  </Route>
);

//React.render(<OrcaApp />, document.getElementById('mount'));
Router.run(AppRoutes, Router.HashLocation, (Root) => {
  React.render(<Root />, document.body);
});
