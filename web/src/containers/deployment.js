import React, { Component } from 'react'
import { connect } from 'react-redux'
import { createSelector } from 'reselect'
import Select from 'react-select'

import { kubeListDeployments, kubeListTags, kubeRollback, kubeSetTag } from '../actions'
import { kubeDeploymentsSelector } from '../selectors'

import { Loading } from '../components'
import { NotFound } from './notfound'

const HEARTBEAT_CODE = '❤️'
const MAX_MESSAGE_SIZE = 50

@connect(
  createSelector(
    kubeDeploymentsSelector,
    deployments => ({ deployments })
  ), {
    // TODO: update get single deployment
    kubeListDeployments,
    kubeListTags,
    kubeRollback,
    kubeSetTag,
  })
export class Deployment extends Component {
  constructor(props) {
    super(props)
    this.state = {
      setUpdate: false,
      setRollback: false,
      versionTag: '',
      messages: [],
    }
  }
  componentWillMount() {
    const { name } = this.props.match.params
    if (!this.props.deployments) {
      this.props.kubeListDeployments()
    }
    // connect websocket
    const u = new URL(window.PUBLIC_URL)
    let scheme = 'ws:'
    if (u.protocol === 'https:') {
      scheme = 'wss:'
    }
    this.conn = new WebSocket(scheme + '//' + u.host + u.pathname + 'events/kube/' + name)
    this.conn.onopen = this.onConnOpen
    this.conn.onmessage = this.onConnMessage
    this.conn.onclose = this.onConnClose
    this.conn.onerror = this.onConnError
  }
  componentWillUnmount() {
    if (this.conn) {
      this.conn.close()
    }
    if (this.t) {
      clearInterval(this.t)
      this.t = null
    }
  }
  addMessage(msg) {
    let { messages } = this.state
    if (messages.length > 0) {
      const lastMessage = messages[messages.length - 1]
      if (lastMessage.msg === msg) {
        // PASS
      } else {
        messages = [ ...messages, { id: lastMessage.id + 1, msg } ]
        if (messages.length > MAX_MESSAGE_SIZE) {
          messages = messages.slice(messages.length - MAX_MESSAGE_SIZE)
        }
        this.setState({ messages })
      }
    } else {
      // first message
      this.setState({ messages: [ { id: 1, msg } ] })
    }
  }
  onConnHeartbeat = () => {
    this.conn.send(HEARTBEAT_CODE)
  }
  onConnOpen = () => {
    this.addMessage('connected')
    this.t = setInterval(this.onConnHeartbeat, 10000)
  }
  onConnMessage = (event) => {
    const ev = JSON.parse(event.data)
    const msg = `[${ev.action}] ${ev.event} replicas: ${ev.status.updatedReplicas}/${ev.status.readyReplicas}/${ev.status.replicas}`
    this.addMessage(msg)
  }
  onConnClose = () => {
    this.addMessage('connection closed')
    this.conn = null
    if (this.t) {
      clearInterval(this.t)
      this.t = null
    }
  }
  onConnError = () => {
    this.addMessage('connection error')
    //this.conn = null
  }
  onUpdateClick = () => {
    const { name } = this.props.match.params
    this.props.kubeListTags(name)
    this.setState({ setUpdate: true })
  }
  onRollbackClick = () => {
    this.setState({ setRollback: true })
  }
  onChange = (value, { action }) => {
    switch (action) {
    case 'select-option':
      this.setState({ versionTag: value.value })
      break
    case 'clear':
      this.setState({ versionTag: '' })
      break
    }
  }
  onUpdateConfirmClick = () => {
    const { name } = this.props.match.params
    const { versionTag } = this.state
    if (versionTag) {
      this.props.kubeSetTag(name, versionTag)
      this.setState({ setUpdate: false })
    }
  }
  onRollbackConfirmClick = () => {
    const { name } = this.props.match.params
    this.props.kubeRollback(name)
    this.setState({ setRollback: false })
  }
  onCancelClick = () => {
    this.setState({ setUpdate: false, setRollback: false })
  }
  render() {
    const { name } = this.props.match.params
    const { deployments } = this.props
    if (!deployments) {
      return (
        <div id="Deployment">
          <Loading />
        </div>
      )
    }
    let dp = null
    for (let i = 0; i < deployments.length; i++) {
      if (deployments[i].name === name) {
        dp = deployments[i]
        break
      }
    }
    if (!dp) {
      return <NotFound />
    }
    const { setUpdate, setRollback, messages } = this.state
    return (
      <div id="Deployment">
        <h2>{ dp.name }</h2>
        <p>Image: { dp.image }</p>
        <p>Replicas: { dp.replicas }</p>
        <p>Revision: { dp.revision }</p>
        <br />
        {
          !setUpdate && !setRollback ? (
            <div className="actions">
              <button className="btn btn-large btn-red update-btn" onClick={this.onUpdateClick}>Update</button>
              <button className="btn btn-large rollback-btn" onClick={this.onRollbackClick}>Rollback</button>
            </div>
          ) : undefined
        }
        {
          setUpdate ? (
            <div className="actions set-update">
              <Select
                isClearable={true}
                isSearchable={true}
                name="version_tag"
                className="tags-select"
                onChange={this.onChange}
                options={[{
                  label: dp.image_name,
                  options: dp.image_tags ? dp.image_tags.map((tag) => ({ value: tag, label: tag })) : [],
                }]}
              />
              <button className="btn btn-large btn-red update-btn" onClick={this.onUpdateConfirmClick}>Update</button>
              <button className="btn btn-large cancel-btn" onClick={this.onCancelClick}>Cancel</button>
            </div>
          ) : undefined
        }
        {
          setRollback ? (
            <div className="actions set-rollback">
              <button className="btn btn-large btn-red rollback-btn" onClick={this.onRollbackConfirmClick}>Confirm Rollback</button>
              <button className="btn btn-large cancel-btn" onClick={this.onCancelClick}>Cancel</button>
            </div>
          ) : undefined
        }
        <div className="message-list">
          <h4>Logs:</h4>
        {
          messages.map((m) => (
            <div key={m.id} className="message">
              { m.msg }
            </div>
          ))
        }
        </div>
      </div>
    )
  }
}
