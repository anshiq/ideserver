const WebSocket = require("ws");
const url = require("url");
const jwt = require("jsonwebtoken");
const { grpcService } = require("./grpcHandler");
const { Service } = require("../models/containerSchema");
require("dotenv").config();
const jwtSecret = process.env.JWTSECRET || "";
async function sendServiceStatus(userId, serviceId) {
  const service = await Service.findById(serviceId);
  if(service ===null) return  webSocketService.sendWsMessageToUser(
    JSON.stringify({ status: "No Service of this name Found" }),
    userId
  );
  if (service.status === "active") {
    return webSocketService.sendWsMessageToUser(
      JSON.stringify({ status: "active" }),
      userId
    );
  } else if (service.status === "spawning") {
    if (webSocketService.activeRequests.has(serviceId))
      return webSocketService.sendWsMessageToUser(
        JSON.stringify({ status: "spawning" }),
        userId
      );
    const statusStream = grpcService.getContainerStatus({
      hostname: serviceId,
    });
    webSocketService.activeRequests.add(serviceId);
    statusStream.on("data", (data) => {
      console.log("Container status update:", data.status);
      Service.findByIdAndUpdate(serviceId, { status: "active" });
      webSocketService.sendWsMessageToUser(
        JSON.stringify({ status: "active" }),
        userId
      );
      webSocketService.activeRequests.delete(serviceId);
    });
    statusStream.on("error", (error) => {
      console.error("Stream error:", error);
      webSocketService.activeRequests.delete(serviceId);
    });

    statusStream.on("end", () => {
      console.log("Stream ended");
      webSocketService.activeRequests.delete(serviceId);
    });
  } else if (service.status === "terminated") {
    return webSocketService.sendWsMessageToUser(
      JSON.stringify({ status: "terminated" }),
      userId
    );
  }
}
class WebSocketService {
  constructor() {
    if (!WebSocketService.instance) {
      console.log("Ws Service Instance created...");
      this.wss = new WebSocket.Server({ noServer: true });
      this.wsClients = new Map();
      this.activeRequests = new Set();
      this.wss.on("connection", (ws, request) =>
        this.handleConnection(ws, request)
      );
      WebSocketService.instance = this; // Set the instance property
    }
    return WebSocketService.instance;
  }

  handleConnection(ws, request) {
    console.log("🔗 New WebSocket connection established");
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
          if(data.type === "userActivePolling") {
            console.log("Pod timeout updated")
          } 
          if(data.type==="serviceStatus"){
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
module.exports = { webSocketService };
