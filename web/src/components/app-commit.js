import React, { Component } from 'react'
import { connect } from 'react-redux'
import moment from 'moment'
import classNames from 'classnames'

import { INSTANCE_STATUSES } from '../api'
import { publishAppConfig, rollbackAppConfig } from '../actions'

import { GravatarImg } from '../components'

function formatConfig(config) {
  return `config id=${config.id} version=${config.version} host=${config.host} instance_id=${config.instance_id}`
}

@connect(null, {
  publishAppConfig,
  rollbackAppConfig,
})
export class AppCommit extends Component {
  constructor(props) {
    super(props)
    this.state = {
      host: '',
      instanceId: '',
      version: '',
      loading: false,
    }
    this.onHostChange = this.onInputChange.bind(this, 'host')
    this.onInstanceIdChange = this.onInputChange.bind(this, 'instanceId')
    this.onVersionChange = this.onInputChange.bind(this, 'version')
  }
  onInputChange(type, e) {
    this.setState({
      [type]: e.target.value.trim(),
    })
  }
  onPublishClick = () => {
    const { appId, commit } = this.props
    const { host, instanceId, version } = this.state
    if (!host || !instanceId || !version) {
      // TODO: replace with modal
      alert('Please fill the form completely')
      return
    }
    this.setState({ loading: true })
    this.props.publishAppConfig(appId, commit.commit_id, host, instanceId, version)
      .then((action) => {
        if (action.error) {
          alert(action.payload.message)
        } else {
          this.setState({ host: '', instanceId: '', version: '' })
        }
        this.setState({ loading: false })
      })
  }
  onRollbackClick(configId) {
    const { appId } = this.props
    if (confirm('Are you really want to rollback this config?')) {
      this.props.rollbackAppConfig(appId, configId)
    }
  }
  render() {
    const { configs, commit, extend, instances } = this.props
    const { host, instanceId, version, loading } = this.state
    return (
      <div className="commit">
        <div className={classNames('commit-wrap', { extend })}>
          <span className="branch">{ commit.branch }</span>
          <span className="commit-id">{ commit.commit_id }</span>
          <div className="message">{ commit.message }</div>
          <span className="author" title={commit.author.email}>
            <GravatarImg email={commit.author.email} />
            { commit.author.name }
            <span className="time" title={commit.author.when}>
              at { moment(commit.author.when).fromNow() }
            </span>
          </span>
          <div className="actions">
            <span className="btn btn-red" onClick={this.props.onActionPublishClick}>P</span>
          </div>
          <div className="form panel-publish">
            <div className="row">
              <label htmlFor="host">Host: </label>
              <input name="host" type="text" placeholder="*" value={host} onChange={this.onHostChange} />
            </div>
            <div className="row">
              <label htmlFor="instance_id">Instance ID: </label>
              <input name="instance_id" type="text" placeholder="*" value={instanceId} onChange={this.onInstanceIdChange} />
            </div>
            <div className="row">
              <label htmlFor="version">Version: </label>
              <input name="version" type="text" placeholder="0" value={version} onChange={this.onVersionChange} />
            </div>
            <span className={classNames('btn', 'btn-red', 'btn-large', { loading })}
                  onClick={this.onPublishClick}>Publish</span>
          </div>
        </div>
        {configs && configs.length > 0 && (
          <div className="configs">
          {configs.map((config) => (
            <span key={config.id} className="config" title={formatConfig(config)}>
              <span className="version">{ config.version }</span>
              <span className="host">{ config.host }</span>
              <span className="instance-id">{ config.instance_id }</span>
              <span className="btn btn-red"
                    title="rollback"
                    onClick={this.onRollbackClick.bind(this, config.id)}>×</span>
            </span>
          ))}
          </div>
        )}
        {instances && instances.length > 0 && (
          <div className="instances">
          {instances.map((instance, i) => {
            const config = configs.find((config) => config.id === instance.config_id)
            return (
              <span key={i} className="instance" title={'belong to: ' + formatConfig(config)}>
                <span className={classNames('status', INSTANCE_STATUSES[instance.status])}
                      title={INSTANCE_STATUSES[instance.status]}></span>
                <span className="host">{ instance.host }</span>
                <span className="instance-id">{ instance.instance_id }</span>
              </span>
            )
          })}
          </div>
        )}
      </div>
    )
  }
}
