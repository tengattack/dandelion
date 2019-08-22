import { RSAA } from 'redux-api-middleware'
import querystring from 'querystring'

import { API_URL, apiPayload } from '../api'

export const APP_SYNC_REQUEST = 'APP_SYNC_REQUEST'
export const APP_SYNC_SUCCESS = 'APP_SYNC_SUCCESS'
export const APP_SYNC_FAILURE = 'APP_SYNC_FAILURE'

export function syncApp(appId = '') {
  return {
    [RSAA]: {
      method: 'POST',
      endpoint: API_URL + '/sync' + (appId ? '/' + appId : ''),
      types: [
        APP_SYNC_REQUEST,
        { type: APP_SYNC_SUCCESS, payload: apiPayload, meta: { appId } },
        { type: APP_SYNC_FAILURE, payload: apiPayload },
      ],
      credentials: 'include',
    }
  }
}

export const APP_LIST_REQUEST = 'APP_LIST_REQUEST'
export const APP_LIST_SUCCESS = 'APP_LIST_SUCCESS'
export const APP_LIST_FAILURE = 'APP_LIST_FAILURE'

export function listAppAppIds() {
  return {
    [RSAA]: {
      method: 'GET',
      endpoint: API_URL + '/list',
      types: [
        APP_LIST_REQUEST,
        { type: APP_LIST_SUCCESS, payload: apiPayload },
        APP_LIST_FAILURE
      ],
      credentials: 'include',
    }
  }
}

export const APP_LIST_CONFIGS_REQUEST = 'APP_LIST_CONFIGS_REQUEST'
export const APP_LIST_CONFIGS_SUCCESS = 'APP_LIST_CONFIGS_SUCCESS'
export const APP_LIST_CONFIGS_FAILURE = 'APP_LIST_CONFIGS_FAILURE'

export function listAppConfigs(appId) {
  return {
    [RSAA]: {
      method: 'GET',
      endpoint: API_URL + '/list/' + appId + '/configs',
      types: [
        APP_LIST_CONFIGS_REQUEST,
        { type: APP_LIST_CONFIGS_SUCCESS, payload: apiPayload },
        APP_LIST_CONFIGS_FAILURE
      ],
      credentials: 'include',
    }
  }
}

export const APP_LIST_COMMITS_REQUEST = 'APP_LIST_COMMITS_REQUEST'
export const APP_LIST_COMMITS_SUCCESS = 'APP_LIST_COMMITS_SUCCESS'
export const APP_LIST_COMMITS_FAILURE = 'APP_LIST_COMMITS_FAILURE'

export function listAppCommits(appId) {
  return {
    [RSAA]: {
      method: 'GET',
      endpoint: API_URL + '/list/' + appId + '/commits',
      types: [
        APP_LIST_COMMITS_REQUEST,
        { type: APP_LIST_COMMITS_SUCCESS, payload: apiPayload },
        APP_LIST_COMMITS_FAILURE
      ],
      credentials: 'include',
    }
  }
}

export const APP_LIST_INSTANCES_REQUEST = 'APP_LIST_INSTANCES_REQUEST'
export const APP_LIST_INSTANCES_SUCCESS = 'APP_LIST_INSTANCES_SUCCESS'
export const APP_LIST_INSTANCES_FAILURE = 'APP_LIST_INSTANCES_FAILURE'

export function listAppInstances(appId) {
  return {
    [RSAA]: {
      method: 'GET',
      endpoint: API_URL + '/list/' + appId + '/instances',
      types: [
        APP_LIST_INSTANCES_REQUEST,
        { type: APP_LIST_INSTANCES_SUCCESS, payload: apiPayload },
        APP_LIST_INSTANCES_FAILURE
      ],
      credentials: 'include',
    }
  }
}

export const APP_PUBLISH_CONFIG_REQUEST = 'APP_PUBLISH_CONFIG_REQUEST'
export const APP_PUBLISH_CONFIG_SUCCESS = 'APP_PUBLISH_CONFIG_SUCCESS'
export const APP_PUBLISH_CONFIG_FAILURE = 'APP_PUBLISH_CONFIG_FAILURE'

export function publishAppConfig(appId, commitId, host, instanceId, version) {
  return {
    [RSAA]: {
      method: 'POST',
      endpoint: API_URL + '/publish/' + appId,
      types: [
        APP_PUBLISH_CONFIG_REQUEST,
        { type: APP_PUBLISH_CONFIG_SUCCESS, payload: apiPayload },
        { type: APP_PUBLISH_CONFIG_FAILURE, payload: apiPayload },
      ],
      credentials: 'include',
      headers: {
        'Content-Type' : 'application/x-www-form-urlencoded',
      },
      body: querystring.stringify({
        commit_id: commitId,
        host,
        instance_id: instanceId,
        version,
      }),
    }
  }
}

export const APP_ROLLBACK_CONFIG_REQUEST = 'APP_ROLLBACK_CONFIG_REQUEST'
export const APP_ROLLBACK_CONFIG_SUCCESS = 'APP_ROLLBACK_CONFIG_SUCCESS'
export const APP_ROLLBACK_CONFIG_FAILURE = 'APP_ROLLBACK_CONFIG_FAILURE'

export function rollbackAppConfig(appId, configId) {
  return {
    [RSAA]: {
      method: 'POST',
      endpoint: API_URL + '/rollback/' + appId,
      types: [
        APP_ROLLBACK_CONFIG_REQUEST,
        { type: APP_ROLLBACK_CONFIG_SUCCESS, payload: apiPayload, meta: { configId } },
        { type: APP_ROLLBACK_CONFIG_FAILURE, payload: apiPayload },
      ],
      credentials: 'include',
      headers: {
        'Content-Type' : 'application/x-www-form-urlencoded',
      },
      body: querystring.stringify({
        id: configId,
      }),
    }
  }
}
