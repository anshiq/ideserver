const { Router } = require("express");
const {
  forgotPassword,
  loginUser,
  signupUser,
  verifyEmailToken,
  verifyForgotPasswordToken,
} = require("../controller/userAuth");

const userRouter = Router();
userRouter.route("/signup").post(signupUser);
userRouter.route("/login").post(loginUser);
userRouter.route("/verify-user").post(verifyEmailToken);
userRouter.route("/forgot-password").post(forgotPassword);
userRouter.route("/verify-forgot-token").post(verifyForgotPasswordToken);

module.exports= { userRouter };
