import React, { Component } from 'react'
import { connect } from 'react-redux'
import { createSelector } from 'reselect'
import Select from 'react-select'

import { kubeListDeployments, kubeListTags, kubeRollback, kubeSetTag } from '../actions'
import { kubeDeploymentsSelector } from '../selectors'

import { Loading } from '../components'
import { NotFound } from './notfound'

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
    }
  }
  componentWillMount() {
    if (!this.props.deployments) {
      this.props.kubeListDeployments()
    }
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
    const { setUpdate, setRollback } = this.state
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
                noOptionsMessage={"Loading..."}
                options={dp.image_tags ? dp.image_tags.map((tag) => ({ value: tag, label: tag })) : []}
              />
              <button className="btn btn-large btn-red update-btn" onClick={this.onUpdateConfirmClick}>Update</button>
            </div>
          ) : undefined
        }
        {
          setRollback ? (
            <div className="actions set-rollback">
              <button className="btn btn-large btn-red rollback-btn" onClick={this.onRollbackConfirmClick}>Confirm Rollback</button>
            </div>
          ) : undefined
        }
      </div>
    )
  }
}
