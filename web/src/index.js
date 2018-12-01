import React from 'react'
import ReactDOM from 'react-dom'
import { createStore, applyMiddleware, compose } from 'redux'
import { Provider } from 'react-redux'
import { BrowserRouter } from 'react-router-dom'
import thunk from 'redux-thunk'
import { apiMiddleware } from 'redux-api-middleware'
import rootReducer from './reducers'
import App from './app'

const composeEnhancers = window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__ || compose
const store = createStore(
  rootReducer,
  composeEnhancers(
    applyMiddleware(thunk, apiMiddleware),
  ),
)

// base on <base>
const u = new URL(document.head.baseURI)
// remove last slash
const basename = u.pathname.endsWith('/') ? u.pathname.substr(0, u.pathname.length - 1) : u.pathname

ReactDOM.render((
  <Provider store={store}>
    <BrowserRouter basename={basename}>
      <App />
    </BrowserRouter>
  </Provider>
), document.getElementById('root'))
