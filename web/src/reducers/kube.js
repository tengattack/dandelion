import {
  KUBE_LIST_REQUEST,
  KUBE_LIST_SUCCESS,
} from '../actions'

const initialState = {
  deployments: null,
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
  }
  return state
}
