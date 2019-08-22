import { combineReducers } from 'redux'

import { app } from './app'
import { kube } from './kube'

const root = combineReducers({
  app,
  kube,
})

export default root
