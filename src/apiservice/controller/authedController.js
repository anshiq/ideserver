const { Container, Service } = require("../models/containerSchema");
const { User } = require("../models/userSchema");
const { grpcHandlers } = require("../Others/grpcHandler");

const getUserDetails = async (req, res) => {
  try {
    const data = await User.findById(req.userId);
    if (!data) return res.status(404).json({ success: false, message: "User not found" });

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
      return res.status(400).json({ success: false, message: "Container ID and Service name are required." });
    }

    const container = await Container.findById(containerId).session(session);
    if (!container) {
      throw new Error("Container not found.");
    }

    const { yamlCode, stack } = container;
    const service = await Service.create([{ userId, linkedContainer: container._id, status: "spawning", name }], { session });

    if (!service || service.length === 0) {
      throw new Error("Failed to create service entry.");
    }

    const serviceId = service[0]._id.toString();
    const grpcResponse = await grpcHandlers.makeContainer({ stack: stack || "", hostName: serviceId, yamlCode: yamlCode || "" });

    if (!grpcResponse) {
      throw new Error("gRPC service creation failed.");
    }

    await session.commitTransaction();
    res.status(200).json({ success: true, message: "Service creation initiated.", service_id: serviceId });
  } catch (error) {
    await session.abortTransaction();
    res.status(500).json({ success: false, message: "Internal server error: " + error.message });
  } finally {
    session.endSession();
  }
};

const deleteContainerService = async (req, res) => {
  const session = await Service.startSession();
  session.startTransaction();

  try {
    const { serviceId } = req.body;
    const userId = req.userId;

    if (!serviceId) {
      return res.status(400).json({ success: false, message: "Service ID is required." });
    }

    const service = await Service.findById(serviceId).session(session);
    if (!service) {
      throw new Error("Service not found.");
    }

    service.status = "terminated";
    await service.save({ session });

    const grpcResponse = await grpcHandlers.deleteContainer({ hostname: service._id.toString() });

    if (!grpcResponse) {
      throw new Error("gRPC service deletion failed.");
    }

    await session.commitTransaction();
    res.status(200).json({ success: true, message: "Service deletion initiated." });
  } catch (error) {
    await session.abortTransaction();
    res.status(500).json({ success: false, message: "Internal server error: " + error.message });
  } finally {
    session.endSession();
  }
};

const reActivateService = async (req, res) => {
  const session = await Service.startSession();
  session.startTransaction();

  try {
    const { serviceId } = req.body;
    if (!serviceId) {
      return res.status(400).json({ message: "Service ID is required." });
    }

    const service = await Service.findById(serviceId).session(session);
    if (!service) {
      throw new Error("Service not found.");
    }

    const container = await Container.findById(service.linkedContainer).session(session);
    if (!container) {
      throw new Error("Linked container not found.");
    }

    service.status = "spawning";
    await service.save({ session });

    const grpcResponse = await grpcHandlers.makeContainer({ stack: container.stack || "", hostName: serviceId, yamlCode: container.yamlCode || "" });

    if (!grpcResponse) {
      throw new Error("gRPC service reactivation failed.");
    }

    await session.commitTransaction();
    res.status(200).json({ message: "Service reactivated successfully.", service });
  } catch (error) {
    await session.abortTransaction();
    res.status(500).json({ message: "Internal Server Error: " + error.message });
  } finally {
    session.endSession();
  }
};

const allUserService = async (req, res) => {
  try {
    const userId = req.userId;
    const services = await Service.find({ userId });

    if (!services.length) {
      return res.status(404).json({ message: "No services found for this user." });
    }

    res.status(200).json({ success: true, services });
  } catch (error) {
    res.status(500).json({ message: "Internal Server Error: " + error.message });
  }
};

const getUserServiceById = async (req, res) => {
  try {
    const { id: serviceId } = req.params;
    const userId = req.userId;

    const service = await Service.findOne({ _id: serviceId, userId });

    if (!service) {
      return res.status(404).json({ message: "Service not found or unauthorized access." });
    }

    res.status(200).json({ success: true, service });
  } catch (error) {
    res.status(500).json({ message: "Internal Server Error: " + error.message });
  }
};

module.exports = {
  getUserDetails,
  createContainerService,
  deleteContainerService,
  allUserService,
  getUserServiceById,
  reActivateService,
};
