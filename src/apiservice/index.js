const express = require("express");
const dotenv = require("dotenv");
const cors = require("cors");
const { createProxyMiddleware } = require("http-proxy-middleware");
const grpc = require("@grpc/grpc-js")
const protoLoader = require("@grpc/proto-loader")
const { userRouter } = require("./routes/userRoutes");
const { connect_db } = require("./db/connect");
const { verifyToken } = require("./middlewares/authMiddleware");
const { authRouter } = require("./routes/authRoutes");

dotenv.config();
var PROTO_PATH = './genproto/main.proto';
var packageDefinition = protoLoader.loadSync(
  PROTO_PATH,
  {keepCase: true,
   longs: String,
   enums: String,
   defaults: true,
   oneofs: true
  });
var protoDescriptor = grpc.loadPackageDefinition(packageDefinition);

var routeguide = protoDescriptor.routeguide;

const port = process.env.PORT || 8080;
const grpcPort = process.env.GRPC_PORT || 8081
const uri = process.env.MONGOURI || "";
const app = express();
app.use(express.json());
app.use(cors());
app.use("/user", userRouter);
app.use("/auth", verifyToken, authRouter);
const map = new Map();
const getTarget = (req) => {
  const host = req.hostname;
  const subdomain = host.split(".")[0];
  const podDns = map.get(subdomain);
  if (podDns) {
    return podDns;
  } else{

  }
};
const proxyMiddleware = createProxyMiddleware({
  target: "http://default-server.com",
  changeOrigin: true,
  ws: true,
  logLevel: "debug",
  router: getTarget,
});

app.use(verifyToken, proxyMiddleware);

const start = () => {
  try {
    connect_db(uri);
    new routeguide.RouteGuide('localhost:50051', grpc.credentials.createInsecure());
 console.log(grpcClient)
    const server = app.listen(port, () => {
      console.log(`Server listening at http://localhost:${port}`);
    });


    server.on("upgrade", proxyMiddleware.upgrade);
  } catch (error) {
    console.log(error);
  }
};

start();

