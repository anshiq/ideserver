const WebSocket = require("ws");
const url = require("url");
const jwt = require("jsonwebtoken");
const { grpcService } = require("./grpcHandler");
const { Service } = require("../models/containerSchema");
const { serviceWatcher } = require("./serviceWatcher");
require("dotenv").config();
const allowedOrigins = [
  /\.iamanshik\.online$/, // Matches any subdomain of iamanshik.online
  ...Array.from({ length: 6 }, (_, i) => `http://localhost:${3000 + i}`) // localhost:3000-3005
];
const jwtSecret = process.env.JWTSECRET || "";
serviceWatcher.addEventListener("onExpire", (serviceName, time) => {
  console.log(`Timeout Event fired for ${serviceName} after time ${time}`);
  webSocketService.activeRequests.delete(serviceName);
  grpcService
    .deleteContainer({ hostname: serviceName })
    .then((e) => {
      console.log(e);
    })
    .catch((e) => console.log(e));

  Service.findByIdAndUpdate(serviceName, { status: "terminated" }).then(() => {
    console.log(
      "Service Id: ",
      serviceName,
      " Remove from serviceWatcher and turned of.."
    );
  });
});
const activeServiceAndMsgPromise = (serviceId) => {
  console.log("hit promise to get grpc stream");
  return new Promise((resolve, reject) => {
    if (!serviceId) {
      console.error("Error: serviceId is undefined");
      return reject("Service ID is required");
    }

    const statusStream = grpcService.getContainerStatus({
      hostname: serviceId,
    });

    webSocketService.activeRequests.add(serviceId);

    let resolved = false; // Track if promise is already resolved/rejected
    let timeoutId = null;

    // Setup timeout
    timeoutId = setTimeout(() => {
      if (!resolved) {
        console.error("Timeout: No response from gRPC stream.");
        resolved = true;

        // Clean up the stream if possible
        try {
          statusStream.cancel();
        } catch (err) {
          console.error("Error cancelling stream:", err);
        }

        reject("Timeout waiting for gRPC stream");
      }
    }, 240000); // 4 minutes timeout

    statusStream.on("data", async (data) => {
      try {
        console.log("Container status update:", data.status);
        await Service.findByIdAndUpdate(serviceId, { status: "active" });
        serviceWatcher.add(serviceId, Date.now());

        if (!resolved) {
          resolved = true;
          clearTimeout(timeoutId);
          resolve("Success!!");
        }
      } catch (error) {
        console.error("Error updating service status:", error);
        if (!resolved) {
          resolved = true;
          clearTimeout(timeoutId);
          reject("Error updating service status");
        }
      }
    });

    statusStream.on("error", (error) => {
      console.error("Stream error:", error);

      if (!resolved) {
        resolved = true;
        clearTimeout(timeoutId);
        reject("Error in gRPC stream: " + error.message);
      }
    });

    statusStream.on("end", () => {
      console.log("Stream ended");

      if (!resolved) {
        resolved = true;
        clearTimeout(timeoutId);
        resolve("Done process!!");
      }
    });
  });
};

async function sendServiceStatus(userId, serviceId) {
  const service = await Service.findById(serviceId);
  if (service === null)
    return webSocketService.sendWsMessageToUser(
      JSON.stringify({ status: "No Service of this name Found" }),
      userId
    );

  if (service.status === "active") {
    if (!serviceWatcher.has(serviceId)) {
      serviceWatcher.add(service._id, Date.now());
    }
    return webSocketService.sendWsMessageToUser(
      JSON.stringify({ status: "active", serviceId }),
      userId
    );
  } else if (service.status === "spawning") {
    if (webSocketService.activeRequests.has(serviceId)) {
      // form cache
      return webSocketService.sendWsMessageToUser(
        JSON.stringify({ status: "spawning", serviceId }),
        userId
      );
    }
    console.log(userId);
    // Add return statement here to ensure the promise chain completes
    activeServiceAndMsgPromise(serviceId)
      .then((e) => {
        console.log(userId);
        webSocketService.sendWsMessageToUser(
          JSON.stringify({ status: "active", serviceId: serviceId }),
          userId
        );
      })
      .catch((e) => {
        webSocketService.sendWsMessageToUser(
          JSON.stringify({ status: "error", serviceId }),
          userId
        );
      });

    return webSocketService.activeRequests.delete(serviceId); // Only delete here after success
  } else if (service.status === "terminated") {
    return webSocketService.sendWsMessageToUser(
      JSON.stringify({ status: "terminated", serviceId }),
      userId
    );
  }

  // Add a default return for any other status
  return webSocketService.sendWsMessageToUser(
    JSON.stringify({ status: "unknown", message: "Service status is unknown" }),
    userId
  );
}
async function pollingServiceUpdateStatus(userId, serviceId) {
  if (serviceWatcher.has(serviceId)) {
    // if service is currently watching then it point to update the service active time
    serviceWatcher.update(serviceId);
    return webSocketService.sendWsMessageToUser(
      JSON.stringify({ serviceId, status: "updated" }),
      userId
    );
  }
  webSocketService.sendWsMessageToUser(
    JSON.stringify({ serviceId, status: "terminated" }),
    userId
  );
}
class WebSocketService {
  constructor() {
    if (!WebSocketService.instance) {
      console.log("Ws Service Instance created...");
      this.wss = new WebSocket.Server({ noServer: true });
      this.wsClients = new Map();
      this.activeRequests = new Set(); // handles all the spawning requests and hold traffic of multiple requests
      this.wss.on("connection", (ws, request) =>
        this.handleConnection(ws, request)
      );
      WebSocketService.instance = this; // Set the instance property
    }
    return WebSocketService.instance;
  }

  handleConnection(ws, request) {
    console.log("🔗 New WebSocket connection established");
    const origin = request.headers.origin;

    if (!origin || !allowedOrigins.some(pattern => new RegExp(pattern).test(origin))) {
      console.log(`Blocked WebSocket connection from: ${origin}`);
      ws.close(1008, "Not allowed by CORS policy");
      return;
    }
    const token = url.parse(request.url, true).query.token;

    let m = jwt.verify(token, jwtSecret, (err, decoded) => {
      if (err) {
        console.log("❌ Invalid Token, closing connection.", decoded);
        ws.close();
      } else {
        const userId = decoded._id; // Assuming username is unique
        this.wsClients.set(userId, ws);
        ws.on("message", (message) => {
          const data = JSON.parse(message);
          if (data.type === "userActivePolling") {
            console.log("Pod timeout updated");
            pollingServiceUpdateStatus(userId, data.serviceId);
          }
          if (data.type === "serviceStatus") {
            sendServiceStatus(userId, data.serviceId);
          }
        });

        ws.on("close", () => {
          console.log(`❌ Connection closed for ${userId}`);
          this.wsClients.delete(userId);
        });

        ws.on("error", (error) => {
          console.error(`⚠️ WebSocket error for ${userId}:`, error);
        });
      }
    });
  }

  handleUpgrade(request, socket, head) {
    console.log("🛠️ WebSocket upgrade request received");

    if (!request.url?.includes("/ws")) {
      socket.write("HTTP/1.1 400 Bad Request\r\n\r\n");
      socket.destroy();
      return;
    }

    this.wss.handleUpgrade(request, socket, head, (ws) => {
      this.wss.emit("connection", ws, request);
    });
  }

  sendWsMessageToUser(message, userId) {
    const ws = this.wsClients.get(userId);
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ message }));
      console.log(`📤 Sent message to ${userId}: ${message}`);
    } else {
      console.log(`⚠️ User ${userId} is not connected.`);
    }
  }

  getActiveConnectionsCount() {
    return this.wsClients.size;
  }
}

const webSocketService = new WebSocketService();
module.exports = { webSocketService, activeServiceAndMsgPromise };
