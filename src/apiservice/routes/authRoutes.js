const { Router } = require("express");
const { getUserDetails, createContainer } = require("../controller/authedController")
const authRouter = Router();
authRouter.route("/api").get(getUserDetails);
authRouter.route("/container").post(
   createContainer
).patch((req,res)=>{
    return res.send("hi")
}).delete((req,res)=>{
    return res.send("hi")
})
module.exports= { authRouter };
