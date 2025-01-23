import bcrypt from "bcrypt";
import jwt from "jsonwebtoken";
import nodemailer from "nodemailer";
import crypto from "crypto";
const saltRounds = parseInt(process.env.SALTROUNDS , 10) || 10;
// const jwtSecret = process.env.JWTSECRET || "";
const jwtSecret = "secret";
const transporter = nodemailer.createTransport({
  service: "gmail",
  host: "smtp.gmail.com",
  port: 465,
  secure: true,
  auth: {
    user: "anshikthind@gmail.com",
    pass: "pnff bbip tzwr kinx",
  },
});
const hashPassword = async (password) => {
  try {
    const salt = await bcrypt.genSalt(saltRounds);
    const hash = await bcrypt.hash(password, salt);
    return hash;
  } catch (err) {
    console.error(err);
    throw err;
  }
};
const comparePassword = async (password, hashedPassword) => {
  try {
    const match = await bcrypt.compare(password, hashedPassword);
    return match;
  } catch (err) {
    console.error(err);
    throw err;
  }
};
const createJwt = (id) => {
  const token = jwt.sign(
    { _id: id },
    jwtSecret,
    //    {
    //   expiresIn: "24h",
    // }
  );
  return token;
};
const sendVerificationEmail = ( mailOptionsObject) => {
  const mailOptions = {
    ...mailOptionsObject,
    from: "anshikthind@gmail.com", // Sender's email
  };
  transporter.sendMail(mailOptions, (error, info) => {
    if (error) {
      console.log("Email verification error: " + error);
    } else {
      console.log("Email sent: " + info.response);
    }
  });
};
const generateVerificationToken = () => {
  return crypto.randomBytes(32).toString("hex");
};
export {
  generateVerificationToken,
  hashPassword,
  comparePassword,
  createJwt,
  sendVerificationEmail,
};
