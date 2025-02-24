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
const createContainer = async (req, res) => {
  const { containerId } = req.body;
  const userId = req.userId
  // const k = await Container.findById(containerId);
  // const yamlcode = k.yamlCode;
  // const stack = k.stack
  // const l = await Service.create({
  //   userId,
  //   linkedContainer: k,
  //   status: "spawning"
  // })
 const m  = await grpcHandlers.makeContainer({ stack:"", hostName: "l._id", yamlCode: "yamlcode" })
  return res.send('hi')
}
module.exports = { getUserDetails ,createContainer};
