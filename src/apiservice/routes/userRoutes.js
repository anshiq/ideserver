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
userRouter.route("/logout").post((req, res) => {
  res.clearCookie("token");
  res.json({ success: true, data: { msg: "Logged out successfully" } });
});
userRouter.route("/verify-user").post(verifyEmailToken);
userRouter.route("/forgot-password").post(forgotPassword);
userRouter.route("/verify-forgot-token").post(verifyForgotPasswordToken);

module.exports = { userRouter };
