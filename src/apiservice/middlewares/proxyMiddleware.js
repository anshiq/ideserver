const { createProxyMiddleware } = require("http-proxy-middleware");
const { PodDnsObject } = require("../Others/getPodDns");
const proxyMiddleware = createProxyMiddleware({
    target: "http://default-server.com",
    changeOrigin: true,
    ws: true,
    logLevel: "debug",
    router: PodDnsObject.getTargetDns,
  });
  module.exports = { proxyMiddleware };