import {
  KUBE_LIST_REQUEST,
  KUBE_LIST_SUCCESS,
  KUBE_LIST_TAGS_SUCCESS,
  KUBE_SET_TAG_SUCCESS,
  KUBE_ROLLBACK_SUCCESS,
} from '../actions'

const initialState = {
  deployments: null,
}

function findDeploymentIndex(ds, name) {
  for (let i = 0; i < ds.length; i++) {
    if (ds[i].name === name) {
      return i
    }
  }
  return -1
}

export function kube(state = initialState, action) {
  switch (action.type) {
  case KUBE_LIST_REQUEST:
    // clear state
    return initialState
  case KUBE_LIST_SUCCESS:
    return {
      ...state,
      deployments: action.payload.deployments,
    }
  case KUBE_LIST_TAGS_SUCCESS: {
    const { deployments } = state
    const i = findDeploymentIndex(deployments, action.meta.name)
    if (i < 0) {
      break
    }
    const dp = { ...deployments[i], image_tags: action.payload.tags }
    return {
      ...state,
      deployments: [
        ...deployments.slice(0, i),
        dp,
        ...deployments.slice(i + 1),
      ]
    }
  }
  case KUBE_SET_TAG_SUCCESS:
  case KUBE_ROLLBACK_SUCCESS: {
    const { deployments } = state
    const i = findDeploymentIndex(deployments, action.meta.name)
    if (i < 0) {
      break
    }
    return {
      ...state,
      deployments: [
        ...deployments.slice(0, i),
        action.payload.deployment,
        ...deployments.slice(i + 1),
      ]
    }
  }
  }
  return state
}
