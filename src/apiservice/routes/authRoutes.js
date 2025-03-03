const { Router } = require("express");
const {
  getUserDetails,
  createContainerService,
  deleteContainerService,
  allUserService,
  getUserServiceById,
  reActivateService,
  getAllContainersUser,
} = require("../controller/authedController");

const authRouter = Router();
authRouter.route("/api").get(getUserDetails);
authRouter.route("/service").post(createContainerService);
authRouter.get("/get-all-containers", getAllContainersUser);

authRouter.get("/services", allUserService);
authRouter.get("/service/:id", getUserServiceById);
authRouter.patch("/service/:id", reActivateService);
authRouter.delete("/service/:id", deleteContainerService);
authRouter.get("/isLoggedIn", (req, res) => {
  res.status(200).json({ msg: "yup loggedin" });
});
module.exports = { authRouter };
