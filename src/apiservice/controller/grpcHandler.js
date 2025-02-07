require("dotenv").config();
const grpc = require("@grpc/grpc-js");
const protoLoader = require("@grpc/proto-loader");
const path = require('path')
var PROTO_PATH = "../genproto/main.proto";
const GRPC_HOST = process.env.GRPC_PORT || 8081;
class AuthorService {
  constructor(grpcClient) {
    this.client = grpcClient;
  }
  
  MakeContainer = async (requestParams) => {
    
    return new Promise((resolve, reject) => {
      this.client.MakeContainer(requestParams, (error, data) => {
        if (error) {
          reject(error);
        } else {
          resolve(data);
        }
      });
    });
  };
  
  
  getContainerDns = async (id) => {
    const requestParams = { author_id: id };

    return new Promise((resolve, reject) => {
      this.client.getContainerDns(requestParams, (error, data) => {
        if (error) {
          reject(error);
        } else {
          resolve(data);
        }
      });
    });
  };

  deleteContainer = async (id) => {
    const requestParams = { author_id: id };

    return new Promise((resolve, reject) => {
      this.client.deleteContainer(requestParams, (error, data) => {
        if (error) {
          reject(error);
        } else {
          resolve(data);
        }
      });
    });
  };

}

var packageDefinition = protoLoader.loadSync(path.join(__dirname,PROTO_PATH), {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});
var packages = grpc.loadPackageDefinition(packageDefinition);
// console.log(packages.ideserver)
const authorClient = new packages.ideserver.RunnerService(
  GRPC_HOST,
  grpc.credentials.createInsecure()
);
const grpcHandlers = new AuthorService(authorClient);

module.exports = { grpcHandlers };
