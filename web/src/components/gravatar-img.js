import React, { Component } from 'react'
import md5 from 'md5'

function getGravatarUrl(email, s = 40) {
  const hash = md5(email.toLowerCase())
  return `https://www.gravatar.com/avatar/${hash}?s=${s}`
}

export function GravatarImg({ email }) {
  return (
    <img className="gravatar-img" src={getGravatarUrl(email)} alt="G" />
  )
}
