import os from 'os'
import path from 'path'
import webpack from 'webpack'

import ExtractTextPlugin from 'extract-text-webpack-plugin'
import HtmlWebpackPlugin from 'html-webpack-plugin'

/* postcss plugins */
import autoprefixer from 'autoprefixer'
import cssnano from 'cssnano'
import postcssUrl from 'postcss-url'
import postcssSprites from 'postcss-sprites'
import updateRule from './build/sprites'

const DEBUG = process.env.NODE_ENV !== 'production' ? true : false

const publicPath = path.join(__dirname, 'public')
const assetsPath = path.join(__dirname, 'public/assets')
const baseUrl = DEBUG ? 'http://localhost:8081/' : ''

const svgoConfig = {
  plugins: [
    { removeTitle: true },
    { convertColors: { shorthex: false } },
    { convertPathData: true },
  ],
}

let entries = [
  './src/index.js',
  './style/main.scss',
]
let loaders = [
  {
    test: /\.(png|jpg|gif)$/,
    use: {
      loader: 'url-loader',
      options: {
        limit: 5000,
        name: 'images/build/[name].[hash:8].[ext]',
      },
    },
  }, {
    test: /\.svg$/,
    use: [
      {
        loader: 'file-loader',
        options: {
          name: 'images/build/[name].[hash:8].[ext]',
        },
      },
      {
        loader: 'svgo-loader',
        options: svgoConfig,
      },
    ],
  },
]
let plugins = [
  new webpack.optimize.ModuleConcatenationPlugin(),
  new HtmlWebpackPlugin({
    filename: DEBUG ? 'index.html' : '../index.html',
    template: './views/index.html',
    inject: true,
    publicUrl: baseUrl,
  }),
  new webpack.DefinePlugin({
    'process.env': {
      'PUBLIC_URL': JSON.stringify(baseUrl),
      'NODE_ENV': JSON.stringify(DEBUG ? 'development' : 'production'),
    },
  }),
]

const postcssOpts = [
  postcssSprites({
    verbose: false,
    basePath: './public',
    spritePath: './public/assets/images/build/',
    spritesmith: {
      padding: 2,
    },
    svgsprite: {
      shape: {
        spacing: {
          padding: 2,
          box: 'padding',
        },
      },
    },
    filterBy: function (image) {
      if (/^(https?:)?\/\//i.test(image.url)) {
        return Promise.reject()
      } else if (image.url.includes('/icons') || image.url.includes('/svg')) {
        return Promise.resolve()
      }
      return Promise.reject()
    },
    groupBy: function (image) {
      let m = /\/images\/(.*?)\/.*/gi.exec(image.url)
      const groupName = m ? m[1] : 'extra'
      image.retina = false
      image.ratio = 1
      if (groupName) {
        m = /@(\d+)x$/gi.exec(groupName)
        if (m) {
          const ratio = parseInt(m[1])
          image.ratio = ratio
          if (ratio > 1) {
            // setup retina mark
            image.retina = true
          }
        }
      }
      return Promise.resolve(groupName)
    },
    hooks: {
      onSaveSpritesheet: function(opts, spritesheet) {
        // We assume that the groups is not an empty array
        const fileName = spritesheet.groups.join('-')
        // not need to get hash
        // const hash = sha1(spritesheet.image)
        //  + '.' + hash.substr(0, 8)
        return path.join(opts.spritePath, 'sprite-' + fileName + '.' + spritesheet.extension)
      },
      onUpdateRule: updateRule,
    },
  }),
]
const postcssLoader = {
  loader: 'postcss-loader',
  options: {
    plugins: (loader) => postcssOpts,
    sourceMap: DEBUG,
  },
}

let cssRoot = publicPath
if (os.platform() === 'win32') {
  cssRoot = cssRoot.replace(/\\/g, '\\\\')
}

if (DEBUG) {
  const hmrUrl = baseUrl + '__webpack_hmr'
  entries = [
    'babel-polyfill',
    //'react-hot-loader/patch',
    //'webpack-dev-server/client?http://localhost:8081/',
    //'webpack-hot-middleware/client?path=' + hmrUrl,
    //'webpack/hot/only-dev-server',
    ...entries,
  ]
  loaders = [
    ...loaders,
    {
      test: /\.(jsx?|es)$/,
      use: [
        {
          loader: 'babel-loader',
          options: {
            babelrc: false,
            presets: [ '@babel/preset-env', '@babel/preset-react' ],
            plugins: [
              'react-hot-loader/babel',
              '@babel/plugin-proposal-class-properties',
              [ '@babel/plugin-proposal-decorators', { legacy: true } ],
            ],
          },
        },
      ],
      exclude: /node_modules/,
    },
    {
      test: /\.tsx?$/,
      use: [
        {
          loader: 'babel-loader',
          options: {
            babelrc: false,
            presets: [ '@babel/preset-env', '@babel/preset-react' ],
            plugins: [ 'react-hot-loader/babel' ],
          },
        },
        'awesome-typescript-loader',
      ],
      exclude: /node_modules/,
    },
    {
      test: /\.css$/,
      use: [
        'style-loader',
        'css-loader?sourceMap&root=' + cssRoot,
        postcssLoader,
      ],
      exclude: /node_modules/,
    },
    {
      test: /\.scss$/,
      use: [
        'style-loader',
        'css-loader?sourceMap&root=' + cssRoot,
        postcssLoader,
        'sass-loader?sourceMap&sourceMapContents',
      ],
      exclude: /node_modules/,
      //include: path.join(__dirname, 'style/scss/sass'),
    },
  ]
  plugins = [
    ...plugins,
    new webpack.LoaderOptionsPlugin({
      debug: true,
    }),
    new webpack.NamedModulesPlugin(),
    new webpack.HotModuleReplacementPlugin(),
  ]
  postcssOpts.push(autoprefixer())
  // fix protocol relative url in blob url
  postcssOpts.push(postcssUrl({
    url: function (asset) {
      if (asset.url.startsWith('//')) {
        return 'http:' + asset.url
      }
      return asset.url
    },
  }))
} else {
  entries = [
    'babel-polyfill',
    ...entries,
  ]
  const extractCSS = new ExtractTextPlugin('css/bundle.[hash:8].css', {
    allChunks: true,
  })
  loaders = [
    ...loaders,
    {
      test: /\.(jsx?|es)$/,
      use: {
        loader: 'babel-loader',
        options: {
          babelrc: false,
          presets: [ '@babel/preset-env', '@babel/preset-react' ],
          plugins: [
            'react-hot-loader/babel',
            '@babel/plugin-proposal-class-properties',
            [ '@babel/plugin-proposal-decorators', { legacy: true } ],
          ],
        },
      },
      exclude: /node_modules/,
    },
    {
      test: /\.tsx?$/,
      use: [
        {
          loader: 'babel-loader',
          options: {
            babelrc: false,
            presets: [ '@babel/preset-env', '@babel/preset-react' ],
          },
        },
        'awesome-typescript-loader',
      ],
      exclude: /node_modules/,
    },
    {
      test: /\.css$/,
      use: extractCSS.extract({
        fallback: 'style-loader',
        publicPath: '../',
        use: [ 'css-loader?root=' + cssRoot, postcssLoader ],
      }),
      exclude: /node_modules/,
    },
    {
      test: /\.scss$/,
      use: extractCSS.extract({
        fallback: 'style-loader',
        publicPath: '../',
        use: [ 'css-loader?root=' + cssRoot, postcssLoader, 'sass-loader' ],
      }),
      exclude: /node_modules/,
    },
  ]
  plugins = [
    ...plugins,
    extractCSS,
  ]
  postcssOpts.push(cssnano({ autoprefixer: { add: true, browsers: ['> 0%'] } }))
}

let webpackConfig = {
  mode: DEBUG ? 'development' : 'production',
  entry: entries,
  output: {
    path: assetsPath,
    filename: 'js/bundle.[hash:8].js',
    publicPath: baseUrl + 'assets/',
  },
  plugins: plugins,
  resolve: {
    extensions: ['.js', '.es', '.ts', '.tsx', '.scss'],
  },
  module: {
    rules: loaders,
  },
}

if (DEBUG) {
  webpackConfig = {
    ...webpackConfig,
    devtool: 'eval-source-map',
    devServer: {
      hot: true,
      historyApiFallback: {
        index: '/assets/index.html',
      },
      proxy: {
        '/api': 'http://localhost:9012',
        '/events': {
          target: 'http://localhost:9012/',
          ws: true,
        },
      },
    },
  }
}

export default webpackConfig
