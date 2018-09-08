import React, { Component } from 'react'
import { connect } from 'react-redux'
import { createSelector } from 'reselect'
import classNames from 'classnames'

import { listAppConfigs, listAppCommits, listAppInstances, syncApp } from '../actions'
import { appConfigsSelector, appCommitsSelector, appInstancesSelector } from '../selectors'

import { AppActions, AppCommit, Loading } from '../components'

@connect(
  createSelector(
    appConfigsSelector,
    appCommitsSelector,
    appInstancesSelector,
    (configs, commits, instances) => ({ configs, commits, instances })
  ), { listAppCommits, listAppConfigs, listAppInstances, syncApp }
)
export class AppDetails extends Component {
  constructor(props) {
    super(props)
    this.state = {
      extendCommitId: '',
    }
  }
  componentWillMount() {
    const { appId } = this.props.match.params
    this.props.listAppCommits(appId)
    this.props.listAppConfigs(appId)
    this.props.listAppInstances(appId)
  }
  onActionPublishClick(commitId) {
    this.setState({
      extendCommitId: this.state.extendCommitId !== commitId ? commitId : '',
    })
  }
  render() {
    const { appId } = this.props.match.params
    const { commits, configs, instances } = this.props
    const { extendCommitId } = this.state
    return (
      <div id="AppDetails">
        <div className="app-title">{ appId }</div>
        <AppActions appId={appId} />
      {
        commits && configs ? (
          <div className="commits">
          {commits.map((commit) => {
            const commitConfigs = configs
              ? configs.filter((config) => config.commit_id === commit.commit_id)
              : null
            const commitInstances = instances
              ? instances.filter((instance) => instance.commit_id === commit.commit_id)
              : null
            return (
              <AppCommit key={commit.commit_id}
                         extend={commit.commit_id === extendCommitId}
                         appId={appId}
                         commit={commit}
                         configs={commitConfigs}
                         instances={commitInstances}
                         onActionPublishClick={this.onActionPublishClick.bind(this, commit.commit_id)} />
            )
          })}
          </div>
        ) : (
          <Loading />
        )
      }
      </div>
    )
  }
}
