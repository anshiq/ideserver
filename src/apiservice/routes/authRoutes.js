const { Router } = require("express");
const { getUserDetails, createContainerService, deleteContainerService, allUserService, getUserServiceById, reActivateService } = require("../controller/authedController")
const authRouter = Router();
authRouter.route("/api").get(getUserDetails);
authRouter.route("/service").post(
    createContainerService
).patch(
    reActivateService
).delete(deleteContainerService)
authRouter.get("/services", allUserService)
authRouter.get("/service/:id", getUserServiceById)
module.exports = { authRouter };
