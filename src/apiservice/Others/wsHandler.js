const WebSocket = require("ws");
const url = require("url");
const jwt = require("jsonwebtoken");
const { grpcService } = require("./grpcHandler");
const { Service } = require("../models/containerSchema");
require('dotenv').config()
const jwtSecret = process.env.JWTSECRET || "";
function sendServiceStatus(userId, serviceId) {
  const statusStream = grpcService.getContainerStatus({ hostname: serviceId })
  statusStream.on('data', (data) => {
    console.log('Container status update:', data.status);
    Service.findByIdAndUpdate(serviceId, { status: "active" })
    console.log("Hit")
    webSocketService.sendWsMessageToUser(JSON.stringify({status:"active"}),userId)
  });

  statusStream.on('error', (error) => {
    console.error('Stream error:', error);
  });

  statusStream.on('end', () => {
    console.log('Stream ended');
  });
}
class WebSocketService {
  constructor() {
    if (!WebSocketService.instance) {
      console.log("Ws Service Instance created...")
      this.wss = new WebSocket.Server({ noServer: true });
      this.wsClients = new Map();
      this.wss.on("connection", (ws, request) => this.handleConnection(ws, request));
      WebSocketService.instance = this; // Set the instance property
    }
    return WebSocketService.instance;
  }

  handleConnection(ws, request) {
    console.log("🔗 New WebSocket connection established");
    const token = url.parse(request.url, true).query.token;
    let m = jwt.verify(token, jwtSecret,
      (err, decoded) => {
        if (err) {
          console.log("❌ Invalid Token, closing connection.", decoded);
          ws.close();
        } else {
          const userId = decoded._id; // Assuming username is unique
          this.wsClients.set(userId, ws);
          console.log(`✅ WebSocket authenticated: ${userId}`);

          ws.on("message", (message) => {
            console.log(`📩 Received message from ${userId}:`, message.toString());
            const data = JSON.parse(message)
            console.log(data)
            sendServiceStatus(userId, data.serviceId)

          });

          ws.on("close", () => {
            console.log(`❌ Connection closed for ${userId}`);
            this.wsClients.delete(userId);
          });

          ws.on("error", (error) => {
            console.error(`⚠️ WebSocket error for ${userId}:`, error);
          });
        }
      }
    );
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