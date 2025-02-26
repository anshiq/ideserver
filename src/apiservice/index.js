const express = require("express");
const dotenv = require("dotenv");
const cors = require("cors");
const grpc = require("@grpc/grpc-js");
const protoLoader = require("@grpc/proto-loader");
const path = require("path");
const http = require("http")
const { userRouter } = require("./routes/userRoutes");
const { connect_db } = require("./db/connect");
const { verifyToken } = require("./middlewares/authMiddleware");
const { authRouter } = require("./routes/authRoutes");
const { adminRoutes } = require("./routes/adminRoutes");
const { webSocketService } = require("./Others/wsHandler");
const { grpcService } = require("./Others/grpcHandler");
dotenv.config();
const port = process.env.PORT || 8080;
const uri = process.env.MONGOURI || "";
const GRPC_HOST = process.env.GRPC_PORT || 8081;
const PROTO_PATH = "./genproto/main.proto";
const app = express();
const httpServer = http.createServer(app);
app.use(express.json());
app.use(cors());
app.use("/user", userRouter);
app.use("/auth", verifyToken, authRouter);
app.use("/admin", verifyToken, adminRoutes);
app.all("/err/:err", (req, res) => {
  res.status(400).send("Error occurred" + req.params.err);
});

const start = () => {
  try {
    var packageDefinition = protoLoader.loadSync(path.join(__dirname, PROTO_PATH), {
      keepCase: true,
      longs: String,
      enums: String,
      defaults: true,
      oneofs: true,
    });
    var packages = grpc.loadPackageDefinition(packageDefinition);
    const authorClient = new packages.ideserver.RunnerService(
      GRPC_HOST,
      grpc.credentials.createInsecure()
    );
    grpcService.connect(authorClient,GRPC_HOST)
    connect_db(uri);
    const server = httpServer.listen(port, () => {
      console.log(`Server listening at http://localhost:${port}`);
    });
    server.on("upgrade", (request, socket, head) => {
      webSocketService.handleUpgrade(request, socket, head);
    });
    
  } catch (error) {
    console.log(error);
  }
};

start();
