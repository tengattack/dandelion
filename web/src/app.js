import React, { Component } from 'react'
import { Route, Switch } from 'react-router-dom'
import { hot } from 'react-hot-loader'
import { Link } from 'react-router-dom'

import logo from '../images/logo.png'

import { AppDetails, Deployment, Homepage, KubePage, NotFound } from './containers'

class App extends Component {
  render() {
    const envTag = window.ENV ? (
      <span className="env-tag">{window.ENV}</span>
    ) : null
    return (
      <div id="app">
        <header className="app-header">
          <img src={logo} className="app-logo" alt="logo" />
          <Link to="/"><h1 className="app-title">Dandelion</h1>{envTag}</Link>
          <span className="s">&gt;</span>
          <Link to="/kube"><h1 className="app-title">Kube</h1></Link>
        </header>
        <Switch>
          <Route exact path='/' component={Homepage} />
          <Route path='/kube' component={KubePage} />
          <Route path='/a/:appId' component={AppDetails} />
          <Route path='/dp/:name' component={Deployment} />
          <Route component={NotFound} />
        </Switch>
      </div>
    )
  }
}

export default hot(module)(App)
