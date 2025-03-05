const { User } = require("../models/userSchema");
const {
  comparePassword,
  createJwt,
  generateVerificationToken,
  hashPassword,
  sendVerificationEmail,
} = require("../Others/AuthFuntions");
async function signupUser(req, res) {
  try {
    const { name, email, password, mobile } = req.body;
    if (!name || !email || !password || !mobile) {
      return res
        .status(400)
        .json({ success: false, error: "All fields are required." });
    }
    const existingUser = await User.findOne({ email });
    if (existingUser) {
      res.json({ success: false, data: { msg: "User already Exist..." } });
      return;
    }
    const hashedpassword = await hashPassword(password);
    const token = generateVerificationToken();
    const data = await User.create({
      name,
      email,
      password: hashedpassword,
      mobile,
      verifyToken: token,
      verified: false,
    });
    if (data) {
      const verificationLink = `${process.env.FRONTEND_URL}/user/auth/verify-email?token=${token}`;
      const mailoptions = {
        to: data.email,
        subject: "Email Verification",
        text: `Please click the following link to verify your email: ${verificationLink}`,
      };
      sendVerificationEmail(mailoptions);
      res.json({
        success: true,
        data: {
          msg: "User signup successful... Check Mail To verify Email....",
        },
      });
    }
  } catch (error) {
    if (error.name === "ValidationError") {
      return res.status(400).json({ success: false, error: error.message });
    }
    res.status(500).json({ success: false, error: "Internal Server Error" });
    console.error(error);
  }
}
async function loginUser(req, res) {
  try {
    const { email, password } = req.body;
    const data = await User.findOne({ email: email });
    if (data) {
      const isUser = await comparePassword(password, data.password);
      if (isUser) {
        if (data.verified === true) {
          const token = createJwt(data._id.toString());
          return res.json({
            success: true,
            data: { token: token, msg: "Login Successfully !!!" },
          });
        }
        return res.json({
          success: false,
          data: { msg: "Verify Your Email First !!!" },
        });
      } else {
        res.json({ success: false, data: { msg: "Wrong Credentials" } });
      }
    } else {
      res.json({ success: false, data: { msg: "Wrong Credentials" } });
    }
  } catch (error) {
    res.json({
      success: false,
      data: { msg: JSON.stringify({ error: error }) },
    });
    console.log(error);
  }
}
const verifyEmailToken = async (req, res) => {
  const token = req.body.token;
  if (!token) {
    return res.status(400).json({ msg: "Verification token is missing." });
  }
  try {
    const user = await User.findOne({ verifyToken: token });
    console.log("hit1");
    if (!user) {
      console.log("hit2");
      return res.status(200).json({
        success: true,
        data: { msg: "User not found or already verified." },
      });
    }
    user.verified = true;
    user.verifyToken = undefined;
    await user.save(); // Save the updated user record
    console.log("hit3");
    res.status(200).json({
      success: true,
      data: { msg: "User  verification successfully." },
    });
  } catch (error) {
    console.error(error);
    res
      .status(500)
      .json({ msg: "An error occurred during email verification." });
  }
};
async function verifyForgotPasswordToken(req, res) {
  try {
    const { token, password } = req.body;
    console.log(req.body);
    const hashedpassword = await hashPassword(password);
    const data = await User.findOne({ verifyToken: token });
    if (data) {
      console.log(data.password);
      console.log(hashedpassword);
      data.verifyToken = undefined;
      data.verified = true;
      data.password = hashedpassword;
      data.save();
      res.json({
        success: true,
        data: { msg: "password updated successfully" },
      });
    }
  } catch (error) {
    console.log(error);
    res.json({ success: false });
  }
}
async function forgotPassword(req, res) {
  try {
    const email = req.body.email;
    const user = await User.findOne({ email: email });
    if (user) {
      const token = generateVerificationToken();
      user.verifyToken = token;
      console.log(token);
      user.save();
      const verificationLink = `${process.env.FRONTEND_URL}/user/auth/reset-password?token=${token}`;
      const mailoptions = {
        to: user.email,
        subject: "Reset password",
        text: `Please click the following link to Reset Password: ${verificationLink}`,
      };
      sendVerificationEmail(mailoptions);
      res.json({
        success: true,
        data: {
          msg: "Please Check your mail for reset password.",
        },
      });
    }
  } catch (error) {
    res.json({
      success: false,
      data: {
        msg: JSON.stringify({
          success: true,
          data: {
            msg: JSON.stringify({err: error}),
          },
        }),
      },
    });
    console.log(error);
  }
}
module.exports = {
  signupUser,
  forgotPassword,
  loginUser,
  verifyEmailToken,
  verifyForgotPasswordToken,
};
