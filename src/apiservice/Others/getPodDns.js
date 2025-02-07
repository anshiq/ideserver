const { grpcHandlers } = require("../controller/grpcHandler");

class PodDnsMap {
  constructor() {
    this.map = new Map();
    this.timerMap = new Map(); // Assuming timerMap might be needed later
    this.getHostNameFromGrpc = grpcHandlers.findAuthorById;
  }

  async getTargetDns(host) {
    const subdomain = host.split(".")[0];
    
    if (this.map.has(subdomain)) {
      return this.map.get(subdomain);
    }
    
    try {
      const podDns = await this.getHostNameFromGrpc(subdomain);
      this.map.set(subdomain, podDns);
      return podDns;
    } catch (error) {
      console.error(`Error fetching DNS for ${subdomain}:`, error);
      return "/err/No_pod_present_for_this_environment_key"
    }
  }
}
const PodDnsObject = new PodDnsMap()
module.exports = {PodDnsObject};

