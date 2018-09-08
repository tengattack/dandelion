import {
  APP_SYNC_SUCCESS,
  APP_LIST_REQUEST,
  APP_LIST_SUCCESS,
  APP_LIST_CONFIGS_SUCCESS,
  APP_LIST_COMMITS_SUCCESS,
  APP_LIST_INSTANCES_SUCCESS,
  APP_PUBLISH_CONFIG_SUCCESS,
  APP_ROLLBACK_CONFIG_SUCCESS,
} from '../actions'

const initialState = {
  appIds: null,
  configs: null,
  commits: null,
  instances: null,
}

export function app(state = initialState, action) {
  switch (action.type) {
  case APP_LIST_REQUEST:
    // clear state
    return initialState
  case APP_LIST_SUCCESS:
  case APP_SYNC_SUCCESS:
    return {
      ...state,
      appIds: action.payload.app_ids,
    }
  case APP_LIST_CONFIGS_SUCCESS:
    return {
      ...state,
      configs: action.payload.configs,
    }
  case APP_LIST_COMMITS_SUCCESS:
    return {
      ...state,
      commits: action.payload.commits,
    }
  case APP_LIST_INSTANCES_SUCCESS:
    return {
      ...state,
      instances: action.payload.instances,
    }
  case APP_PUBLISH_CONFIG_SUCCESS:
    return {
      ...state,
      configs: [ action.payload.config, ...state.configs ],
    }
  case APP_ROLLBACK_CONFIG_SUCCESS:
    return {
      ...state,
      configs: state.configs.filter((config) => config.id !== action.payload.config.id)
    }
  }
  return state
}
