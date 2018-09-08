import React, { Component } from 'react'
import { getGravatarUrl } from '../api'

export function GravatarImg({ email }) {
  return (
    <img className="gravatar-img" src={getGravatarUrl(email)} alt="G" />
  )
}
