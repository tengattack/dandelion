import React, { Component } from 'react'
import { connect } from 'react-redux'
import { createSelector } from 'reselect'
import { Link } from 'react-router-dom'

import { listAppAppIds } from '../actions'
import { appAppIdsSelector } from '../selectors'

import { AppActions, Loading } from '../components'

@connect(
  createSelector(
    appAppIdsSelector,
    appIds => ({ appIds })
  ), { listAppAppIds }
)
export class Homepage extends Component {
  componentWillMount() {
    this.props.listAppAppIds()
  }
  render() {
    const { appIds } = this.props
    return (
      <div id="Homepage">
        <AppActions />
      {
        appIds ? (
          <div className="app-list">
          {appIds.map((appId) => (
            <Link key={appId} className="item" to={`/a/${appId}`}>
              { appId }
            </Link>
          ))}
          </div>
        ) : (
          <Loading />
        )
      }
      </div>
    )
  }
}
