import React, { Component } from 'react'
import { connect } from 'react-redux'
import classNames from 'classnames'

import { listAppCommits, syncApp } from '../actions'

@connect(null, {
  listAppCommits,
  syncApp,
})
export class AppActions extends Component {
  constructor(props) {
    super(props)
    this.state = {
      syncing: false,
    }
  }
  onSyncClick = () => {
    const { appId } = this.props
    this.setState({ syncing: true })
    this.props.syncApp(appId)
      .then((action) => {
        if (action.error) {
          alert(action.payload.message)
        } else {
          if (appId) {
            // fetch latest commits
            this.props.listAppCommits(appId)
          }
        }
        this.setState({ syncing: false })
      })
  }
  render() {
    const { syncing } = this.state
    return (
      <div className="app-actions">
        <span className={classNames('btn', 'btn-large', 'btn-sync', { loading: syncing })}
              onClick={this.onSyncClick}>
          Sync
        </span>
      </div>
    )
  }
}
