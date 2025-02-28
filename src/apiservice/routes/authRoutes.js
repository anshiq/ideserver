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
authRouter
  .route("/service")
  .post(createContainerService)
  .patch(reActivateService)
  .delete(deleteContainerService);
authRouter.get("/get-all-containers", getAllContainersUser);
authRouter.get("/services", allUserService);
authRouter.get("/service/:id", getUserServiceById);
authRouter.get("/isLoggedIn", (req,res)=>{
    res.status(200).json({msg:"yup loggedin"})
});
module.exports = { authRouter };
