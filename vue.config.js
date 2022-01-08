// vue.config.js
module.exports = {
  // options...
  devServer: {
    disableHostCheck: true
  },
   publicPath: process.env.NODE_ENV === 'production' ?
    '/app' :
    '/'
}