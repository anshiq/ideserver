require('dotenv').config()
const { grpcHandlers } = require("../controller/grpcHandler");

class PodDnsMap {
  constructor() {
    this.map = new Map();
    this.timerMap = new Map(); // Assuming timerMap might be needed later
    this.getHostNameFromGrpc = grpcHandlers.findAuthorById;
  }

  async getTargetDns(host) {
    const subdomain = host.split(".")[0];;
    
    if (this.map.has(subdomain)) {
      return this.map.get(subdomain);
    }
    console.log(typeof host,subdomain); // it is string

    try {
      const data = await grpcHandlers.getContainerDns(subdomain);
      console.log(this.map,data,"hi");
      const podDns = data.podDns;
      this.map.set(subdomain, podDns);
      console.log(podDns.length)
      if(podDns.length== 0) throw new Error("no namespace dns found in cluster")
      return podDns;

    } catch (error) {
      console.error(`Error fetching DNS for ${subdomain}:`, error);
      return process.env.API_SERVICE_PUBLIC_IP +"/err/No_pod_present_for_this_environment_key" +subdomain;
    }
  }
}
const PodDnsObject = new PodDnsMap();
module.exports = { PodDnsObject };
