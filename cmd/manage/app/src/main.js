"use strict";
import React from 'react';
import mui from 'material-ui';
import injectTapEventPlugin from 'react-tap-event-plugin';
import Router, { Route, DefaultRoute, NotFoundRoute, Redirect, Link } from 'react-router';
import 'flexboxgrid/dist/flexboxgrid.css';


injectTapEventPlugin();

// Our application
import OrcaApp from './OrcaApp';
import Home from './components/Home';
import About from './components/About';
import Contact from './components/Contact';

const AppRoutes = (
  <Route path="/" handler={OrcaApp}>
    <DefaultRoute handler={Home} />
    <Route name="about" handler={About} />
    <Route name="contact" handler={Contact} />
  </Route>
);

//React.render(<OrcaApp />, document.getElementById('mount'));
Router.run(AppRoutes, Router.HashLocation, (Root) => {
  React.render(<Root />, document.body);
});
