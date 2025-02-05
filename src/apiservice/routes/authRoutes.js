const { Router } = require("express");
const { getUserDetails } = require("../controller/authedController")
const authRouter = Router();
authRouter.route("/api").get(getUserDetails);
module.exports= { authRouter };
