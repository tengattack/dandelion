import React, { Component } from 'react'
import { connect } from 'react-redux'
import { createSelector } from 'reselect'
import Select from 'react-select'

import { kubeGetDetail, kubeListTags, kubeSetReplicas, kubeRollback, kubeSetTag, kubeRestart } from '../actions'
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
    kubeGetDetail,
    kubeListTags,
    kubeSetReplicas,
    kubeRollback,
    kubeSetTag,
    kubeRestart,
  })
export class Deployment extends Component {
  constructor(props) {
    super(props)
    this.state = {
      loading: true,
      detail: null,
      setUpdate: false,
      setReplicas: false,
      setRollback: false,
      setRestart: false,
      versionTag: '',
      replicasValue: '',
      tags: null,
      messages: [],
    }
  }
  componentWillMount() {
    const { name } = this.props.match.params
    this.props.kubeGetDetail(name)
      .then((res) => {
        this.setState({ detail: res.payload, loading: false })
      })

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
  handleDeploymentUpdate(res) {
    if (res.payload && res.payload.deployment) {
      this.setState({
        detail: {
          ...this.state.detail,
          deployment: res.payload.deployment,
        },
      })
    }
  }
  addMessage(msg) {
    let { messages } = this.state
    if (messages.length > 0) {
      const lastMessage = messages[messages.length - 1]
      messages = [ ...messages, { id: lastMessage.id + 1, msg } ]
      if (messages.length > MAX_MESSAGE_SIZE) {
        messages = messages.slice(messages.length - MAX_MESSAGE_SIZE)
      }
      this.setState({ messages })
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
    if (event.data === HEARTBEAT_CODE) {
      return
    }
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
      .then((res) => {
        this.setState({ tags: res.payload.tags })
      })
    this.setState({ setUpdate: true })
  }
  onSetReplicasClick = () => {
    this.setState({ setReplicas: true })
  }
  onRollbackClick = () => {
    this.setState({ setRollback: true })
  }
  onRestartClick = () => {
    this.setState({ setRestart: true })
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
  onReplicasInputChange = (e) => {
    let val = e.target.value.trim()
    val = parseInt(val)
    if (!val) {
      val = ''
    } else if (val < 0) {
      val = 0
    }
    this.setState({ replicasValue: val })
  }
  onUpdateConfirmClick = () => {
    const { name } = this.props.match.params
    const { versionTag } = this.state
    if (versionTag) {
      this.props.kubeSetTag(name, versionTag)
        .then((res) => {
          this.handleDeploymentUpdate(res)
        })
      this.setState({ setUpdate: false })
    }
  }
  onSetReplicasConfirmClick = () => {
    const { name } = this.props.match.params
    const { replicasValue } = this.state
    if (replicasValue) {
      this.props.kubeSetReplicas(name, replicasValue)
        .then((res) => {
          this.setState({ detail: res.payload })
        })
      this.setState({ setReplicas: false })
    }
  }
  onRollbackConfirmClick = () => {
    const { name } = this.props.match.params
    this.props.kubeRollback(name)
      .then((res) => {
        this.handleDeploymentUpdate(res)
      })
    this.setState({ setRollback: false })
  }
  onRestartConfirmClick = () => {
    const { name } = this.props.match.params
    this.props.kubeRestart(name)
      .then((res) => {
        this.handleDeploymentUpdate(res)
      })
    this.setState({ setRestart: false })
  }
  onCancelClick = () => {
    this.setState({ setUpdate: false, setReplicas: false, setRollback: false, setRestart: false })
  }
  render() {
    const { detail, loading } = this.state
    if (loading) {
      return (
        <div id="Deployment">
          <Loading />
        </div>
      )
    }
    if (!detail || !detail.deployment) {
      return <NotFound />
    }
    const dp = detail.deployment
    const { setUpdate, setReplicas, setRollback, setRestart, tags, messages } = this.state
    return (
      <div id="Deployment">
        <h2>{ dp.name }</h2>
        <p className="image-name">Image: { dp.image }</p>
        <p>Replicas: { dp.replicas }</p>
        {
          detail.hpa ? (
            <p>HPA Replicas: { detail.hpa.min_replicas } - { detail.hpa.max_replicas }</p>
          ) : undefined
        }
        <p>Revision: { dp.revision }</p>
        <br />
        {
          !setUpdate && !setReplicas && !setRollback && !setRestart ? (
            <div className="actions">
              <button className="btn btn-large btn-red update-btn" onClick={this.onUpdateClick}>Update</button>
              <button className="btn btn-large replicas-btn" onClick={this.onSetReplicasClick}>Set Replicas</button>
              <button className="btn btn-large rollback-btn" onClick={this.onRollbackClick}>Rollback</button>
              <button className="btn btn-large restart-btn" onClick={this.onRestartClick}>Restart</button>
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
                  options: tags ? tags.map((tag) => ({ value: tag, label: tag })) : [],
                }]}
              />
              <button className="btn btn-large btn-red update-btn" onClick={this.onUpdateConfirmClick}>Update</button>
              <button className="btn btn-large cancel-btn" onClick={this.onCancelClick}>Cancel</button>
            </div>
          ) : undefined
        }
        {
          setReplicas ? (
            <div className="actions set-replicas">
              <div className="form-input">
                <input type="number" name="replicas" value={this.state.replicasValue} onChange={this.onReplicasInputChange} />
              </div>
              <button className="btn btn-large btn-red replicas-btn" onClick={this.onSetReplicasConfirmClick}>Confirm Set</button>
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
        {
          setRestart ? (
            <div className="actions set-restart">
              <button className="btn btn-large btn-red restart-btn" onClick={this.onRestartConfirmClick}>Confirm Restart</button>
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
