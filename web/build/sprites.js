
import postcss from 'postcss'
import { updateRule } from 'postcss-sprites/lib/core'

function propsToFixed(props) {
  for (const prop in props) {
    const val = props[prop]
    if (val.toString().includes('.')) {
      props[prop] = val.toFixed(5)
    }
  }
}

export default function spritesUpdateRule(rule, token, image) {
  if (image.spriteUrl.endsWith('.svg')) {
    const { spriteUrl, spriteWidth, spriteHeight, ratio, coords } = image
    const props = {
      // 2 is for padding (opts.svgsprite.shape.padding)
      posX: -1 * Math.abs(coords.x / ratio) - 2,
      posY: -1 * Math.abs(coords.y / ratio) - 2,
      sizeX: spriteWidth / ratio,
      sizeY: spriteHeight / ratio,
    }
    propsToFixed(props)
    const backgroundDecl = postcss.decl({
      prop: 'background',
      value: `url(${spriteUrl}) ${props.posX}px ${props.posY}px / ${props.sizeX}px ${props.sizeY}px`,
    })
    rule.insertAfter(token, backgroundDecl)
    return
  }
  updateRule(rule, token, image)
}
