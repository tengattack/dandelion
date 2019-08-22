import React, { Component } from 'react'
import { connect } from 'react-redux'
import { createSelector } from 'reselect'
import { Link } from 'react-router-dom'

import { kubeListDeployments } from '../actions'
import { kubeDeploymentsSelector } from '../selectors'

import { Loading } from '../components'

@connect(
  createSelector(
    kubeDeploymentsSelector,
    deployments => ({ deployments })
  ), { kubeListDeployments }
)
export class KubePage extends Component {
  componentWillMount() {
    this.props.kubeListDeployments()
  }
  render() {
    const { deployments } = this.props
    return (
      <div id="KubePage">
      {
        deployments ? (
          <div className="deployment-list">
          {deployments.map((dp) => (
            <Link key={dp.name} className="item" to={`/dp/${dp.name}`}>
              { dp.name }
            </Link>
          ))}
          </div>
        ) : <Loading />
      }
      </div>
    )
  }
}
