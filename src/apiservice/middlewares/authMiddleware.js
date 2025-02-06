const jwt = require("jsonwebtoken");
require('dotenv').config()
const jwtSecret = process.env.JWTSECRET || "";
function verifyToken(req, res, next) {
  const token = req.cookies?.token;

  if (!token) return res.status(401).json({ error: "Access denied" });
  try {
    const decoded = jwt.verify(token, jwtSecret);
    req.userId = decoded._id;
    next();
  } catch (error) {
    res.status(401).json({ error: "Invalid token" });
  }
}

module.exports = { verifyToken };
