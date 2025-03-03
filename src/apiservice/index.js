const express = require("express");
const dotenv = require("dotenv");
const cors = require("cors");
const grpc = require("@grpc/grpc-js");
const protoLoader = require("@grpc/proto-loader");
const path = require("path");
const http = require("http");
const { userRouter } = require("./routes/userRoutes");
const { connect_db } = require("./db/connect");
const { verifyToken } = require("./middlewares/authMiddleware");
const { authRouter } = require("./routes/authRoutes");
const { adminRoutes } = require("./routes/adminRoutes");
const { webSocketService } = require("./Others/wsHandler");
const { grpcService } = require("./Others/grpcHandler");
const { Service } = require("./models/containerSchema");
const { serviceWatcher } = require("./Others/serviceWatcher");
const allowedOrigins = [
  /\.iamanshik\.online$/, // Matches any subdomain of iamanshik.online
  ...Array.from({ length: 6 }, (_, i) => `http://localhost:${3000 + i}`) // localhost:3000-3005
];
dotenv.config();
const port = process.env.PORT || 8080;
const uri = process.env.MONGOURI || "";
const GRPC_HOST = process.env.GRPC_PORT || 8081;
const PROTO_PATH = "./genproto/main.proto";
const app = express();

app.use(express.json());
app.use(cors({
  origin: "*",
}));
app.use("/user", userRouter);
app.use("/auth", verifyToken, authRouter);
app.use("/admin", verifyToken, adminRoutes);
app.all("/err/:err", (req, res) => {
  res.status(400).send("Error occurred" + req.params.err);
});
const httpServer = http.createServer(app);
const start = async () => {
  try {
    var packageDefinition = protoLoader.loadSync(
      path.join(__dirname, PROTO_PATH),
      {
        keepCase: true,
        longs: String,
        enums: String,
        defaults: true,
        oneofs: true,
      }
    );
    var packages = grpc.loadPackageDefinition(packageDefinition);
    const authorClient = new packages.ideserver.RunnerService(
      GRPC_HOST,
      grpc.credentials.createInsecure()
    );
    grpcService.connect(authorClient, GRPC_HOST);
    connect_db(uri)
      .then(() => {
        Service.find({ status: "active" })
          .then((data) => {
            data.map((each) => {
              serviceWatcher.update(each._id);
            });
            console.log("All Active service added to timeout servicehandle: ")
          })
          .catch((e) => {
            console.log(
              "Failed to Add Services in timeout terminating service.... ",
              e
            );
          });
      })
      .catch((e) => {
        console.log("error connecting db");
        process.exit(1);
      });
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
