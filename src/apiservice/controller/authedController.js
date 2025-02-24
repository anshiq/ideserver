const { Container, Service } = require("../models/containerSchema");
const { User } = require("../models/userSchema");
const { grpcHandlers } = require("../Others/grpcHandler")
const getUserDetails = async (req, res) => {
  try {
    const data = await User.findById(req.userId);
    if (!data) return;
    const user = {
      success: true,
      data: {
        name: data.name,
        mobile: data.mobile,
        email: data.email,
      },
    };
    res.json(user);
  } catch (error) {
    res.json({ success: false, data: { msg: JSON.stringify({ err: error }) } });
  }
};
const createContainerService = async (req, res) => {
  const session = await Service.startSession(); // Start a transaction session
  session.startTransaction();

  try {
    const { containerId,name } = req.body;
    const userId = req.userId;

    if (!containerId) {
      return res.status(400).json({ success: false, message: "Container ID is required." });
    }

    if (!userId) {
      return res.status(403).json({ success: false, message: "Unauthorized: User ID is missing." });
    }

    // Find container by ID
    const container = await Container.findById(containerId);
    if (!container) {
      return res.status(404).json({ success: false, message: "Container not found." });
    }

    const { yamlCode, stack } = container;

    // Create a service entry within the transaction
    const service = await Service.create([{ 
      userId, 
      linkedContainer: container._id, 
      status: "spawning", 
      name :name
    }], { session });

    if (!service || service.length === 0) {
      throw new Error("Failed to create service entry.");
    }

    const serviceId = service[0]._id.toString();

    // Call gRPC to create the container
    const grpcResponse = await grpcHandlers.makeContainer({
      stack: stack || "",
      hostName: serviceId,
      yamlCode: yamlCode || "",
    });

    if (!grpcResponse) {
      throw new Error("gRPC container creation failed.");
    }

    // Commit transaction if everything is successful
    await session.commitTransaction();
    session.endSession();

    return res.status(200).json({ success: true, message: "Container creation initiated." });

  } catch (error) {
    console.error("Error in createContainerService:", error);

    // Rollback transaction in case of an error
    await session.abortTransaction();
    session.endSession();

    return res.status(500).json({ success: false, message: "Internal server error. " + error.message });
  }
};


module.exports = { getUserDetails ,createContainerService};
