import { ApiError as ApiErrorA } from 'redux-api-middleware'

export const PUBLIC_URL = window.PUBLIC_URL || '/'
// PUBLIC_URL has last slash
export const API_URL = PUBLIC_URL + 'api/v1'

export const INSTANCE_STATUSES = [ 'offline', 'checking', 'syncing', 'online', 'error' ]

class ApiError extends ApiErrorA {
  constructor(status, statusText, response) {
    super(status, statusText, response)
    this.message = statusText
  }
}

export function apiPayload(action, state, res) {
  return res.json().then((json) => {
    if (json.code !== 0) {
      throw new ApiError(json.code, json.info, json)
    } else {
      return json.info
    }
  })
}
