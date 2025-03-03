const { Container, Service } = require("../models/containerSchema");
const { User } = require("../models/userSchema");
const { grpcService } = require("../Others/grpcHandler");
const { serviceWatcher } = require("../Others/serviceWatcher");
const {  activeServiceAndMsgPromise, webSocketService } = require("../Others/wsHandler");

const getUserDetails = async (req, res) => {
  try {
    const data = await User.findById(req.userId);
    if (!data)
      return res
        .status(404)
        .json({ success: false, message: "User not found" });

    res.status(200).json({
      success: true,
      data: {
        name: data.name,
        mobile: data.mobile,
        email: data.email,
      },
    });
  } catch (error) {
    res.status(500).json({ success: false, message: "Internal Server Error" });
  }
};

const createContainerService = async (req, res) => {
  const session = await Service.startSession();
  session.startTransaction();

  try {
    const { containerId, name } = req.body;
    const userId = req.userId;

    if (!containerId || !name) {
      return res.status(400).json({
        success: false,
        message: "Container ID and Service name are required.",
      });
    }

    const container = await Container.findById(containerId).session(session);
    if (!container) {
      throw new Error("Container not found.");
    }

    const { yamlCode, stack } = container;
    const service = await Service.create(
      [{ userId, linkedContainer: container._id, status: "spawning", name }],
      { session }
    );

    if (!service || service.length === 0) {
      throw new Error("Failed to create service entry.");
    }

    const serviceId = service[0]._id.toString();
    const grpcResponse = await grpcService.makeContainer({
      stack: stack || "",
      hostName: serviceId,
      yamlCode: yamlCode || "",
    });

    if (!grpcResponse) {
      throw new Error("gRPC service creation failed.");
    }

    await session.commitTransaction();
    console.log("hit",serviceId)
    // activeServiceAndMsgPromise(serviceId).then(e=>{
    //    webSocketService.activeRequests.delete(serviceId);
    //     console.log("Deleted from createContainerService controller: the activeRequest of ws")
    // }).catch(e=>{
    //    webSocketService.activeRequests.delete(serviceId);
    //     console.log("Deleted from createContainerService controller: the activeRequest of ws")
    // })
    res.status(200).json({
      success: true,
      message: "Service creation initiated.",
      service_id: serviceId,
    });
  } catch (error) {
    await session.abortTransaction();
    res.status(500).json({
      success: false,
      message: "Internal server error: " + error.message,
    });
  } finally {
    session.endSession();
  }
};

const deleteContainerService = async (req, res) => {
  const session = await Service.startSession();
  session.startTransaction();

  try {
    const { id: serviceId } = req.params;
    const userId = req.userId;

    if (!serviceId) {
      return res
        .status(400)
        .json({ success: false, message: "Service ID is required." });
    }

    const service = await Service.findById(serviceId).session(session);
    if (!service) {
      throw new Error("Service not found.");
    }

    service.status = "terminated";
    await service.save({ session });

    const grpcResponse = await grpcService.deleteContainer({
      hostname: service._id.toString(),
    });

    if (!grpcResponse) {
      throw new Error("gRPC service deletion failed.");
    }

    await session.commitTransaction();
    res
      .status(200)
      .json({ success: true, message: "Service deletion initiated." });
  } catch (error) {
    console.log(error)
    await session.abortTransaction();
    res.status(500).json({
      success: false,
      message: "Internal server error: " + error.message,
    });
  } finally {
    session.endSession();
  }
};

const reActivateService = async (req, res) => {
  const session = await Service.startSession();
  session.startTransaction();

  try {
    const { id: serviceId } = req.params;
    if (!serviceId) {
      return res.status(400).json({ message: "Service ID is required." });
    }

    const service = await Service.findById(serviceId).session(session);
    if (!service) {
      throw new Error("Service not found.");
    }

    const container = await Container.findById(service.linkedContainer).session(
      session
    );
    if (!container) {
      throw new Error("Linked container not found.");
    }

    service.status = "spawning";
    await service.save({ session });

    const grpcResponse = await grpcService.makeContainer({
      stack: container.stack || "",
      hostName: serviceId,
      yamlCode: container.yamlCode || "",
    });

    if (!grpcResponse) {
      throw new Error("gRPC service reactivation failed.");
    }

    await session.commitTransaction();

  //   activeServiceAndMsgPromise(service._id).then(e=>{
  //     console.log("Deleted from activeservice controller: the activeRequest of ws")
  //     webSocketService.activeRequests.delete(serviceId);
  //  }).catch(e=>{
  //   console.log("Deleted from activeservice controller: the activeRequest of ws")
  //     webSocketService.activeRequests.delete(serviceId);
  //  })
  return  res
      .status(200)
      .json({ message: "Service reactivated successfully.", service });
  } catch (error) {
    await session.abortTransaction();
    console.log(error)
  return  res
      .status(500)
      .json({ message: "Internal Server Error hit: " + error.message });
  } finally {
    session.endSession();
  }
};

const allUserService = async (req, res) => {
  try {
    const userId = req.userId;
    const services = await Service.find({ userId });

    if (!services.length) {
      return res
        .status(404)
        .json({ message: "No services found for this user." });
    }

    res.status(200).json({ success: true, services });
  } catch (error) {
    res
      .status(500)
      .json({ message: "Internal Server Error: " + error.message });
  }
};

const getUserServiceById = async (req, res) => {
  try {
    const { id: serviceId } = req.params;
    const userId = req.userId;

    if (!serviceId || !userId) {
      return res.status(400).json({ message: "Invalid service ID or user ID." });
    }

    const service = await Service.findOne({ _id: serviceId, userId });
    if (!service) {
      return res.status(404).json({ message: "Service not found or unauthorized access." });
    }

    const container = await Container.findById(service.linkedContainer);
    if (!container) {
      return res.status(404).json({ message: "Linked container not found." });
    }

    res.status(200).json({ success: true, service, container });
  } catch (error) {
    console.error("Error in getUserServiceById:", error);
    res.status(500).json({ message: "Internal Server Error: " + error.message });
  }
};
const getAllContainersUser = async (req, res) => {
  try {
    const containers = await Container.find().select([
      "_id",
      "name",
      "stack",
      "iconUrl",
    ]);
    res.json(containers);
  } catch (error) {
    res.status(500).json({ error: "Server error", details: error.message });
  }
};

module.exports = {
  getUserDetails,
  createContainerService,
  deleteContainerService,
  allUserService,
  getUserServiceById,
  reActivateService,
  getAllContainersUser,
};
