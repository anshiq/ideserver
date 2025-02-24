const { Router } = require("express");
const { getUserDetails, createContainerService } = require("../controller/authedController")
const authRouter = Router();
authRouter.route("/api").get(getUserDetails);
authRouter.route("/container").post(
   createContainerService
).patch((req,res)=>{
    return res.send("hi")
}).delete((req,res)=>{
    return res.send("hi")
})
module.exports= { authRouter };
