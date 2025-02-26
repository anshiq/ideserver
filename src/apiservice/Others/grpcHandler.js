class GrpcService {
  constructor() {
    if (!GrpcService.instance) {
      this._client = null;
      GrpcService.instance = this;
      console.log("🔄 Grpc Service Instance Created...");
    }
    return GrpcService.instance;
  }

  connect(authorClient, GRPC_HOST) {
    if (!authorClient || typeof authorClient !== "object") {
      throw new Error("❌ Invalid gRPC client provided.");
    }
    if (!this._client) {
      this._client = authorClient;
      console.log("✅ gRPC service connected. at: " + GRPC_HOST);
    } else {
      console.log("⚡ gRPC service already configured.");
    }
  }

  get client() {
    if (!this._client) {
      throw new Error("❌ gRPC client not connected. Call `connect()` first.");
    }
    return this._client;
  }

  async makeContainer({ stack, hostName, yamlCode }) {
    const requestParams = { techStack: stack, yamlFileCode: yamlCode, hostname: hostName };
    
    return new Promise((resolve, reject) => {
      this.client.makeContainer(requestParams, (error, data) => {
        if (error) reject(error);
        else resolve(data);
      });
    });
  }

  async deleteContainer({ hostname }) {
    const requestParams = { hostname };
    
    return new Promise((resolve, reject) => {
      this.client.deleteContainer(requestParams, (error, data) => {
        if (error) reject(error);
        else resolve(data);
      });
    });
  }

  getContainerStatus({ hostname }) {
    const requestParams = { hostname };
    
    // Since this is a streaming response, we don't wrap it in a Promise
    // Instead, we return the stream so the caller can attach event handlers
    try {
      const stream = this.client.getContainerStatus(requestParams);
      console.log(`⚡ Streaming container status for hostname: ${hostname}`);
      return stream;
    } catch (error) {
      console.error(`❌ Error initiating container status stream: ${error.message}`);
      throw error;
    }
  }
}

// Export a single instance (Singleton)
const grpcService = new GrpcService();
// Object.freeze(grpcService) // freezes the object for further modifications
module.exports = {grpcService};