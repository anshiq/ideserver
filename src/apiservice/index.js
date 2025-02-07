const express = require("express");
const dotenv = require("dotenv");
const cors = require("cors");
const { userRouter } = require("./routes/userRoutes");
const { connect_db } = require("./db/connect");
const { verifyToken } = require("./middlewares/authMiddleware");
const { authRouter } = require("./routes/authRoutes");
const { proxyMiddleware  } = require("./middlewares/proxyMiddleware") 
dotenv.config();
const port = process.env.PORT || 8080;
const uri = process.env.MONGOURI || "";
const app = express();
app.use(express.json());
app.use(cors());
app.use("/user", userRouter);
app.use("/auth", verifyToken, authRouter);
app.all("/err/:err", (req, res) => {
  res.status(400).send("Error occurred" + req.params.err);
});
app.use(verifyToken, proxyMiddleware);

const start = () => {
  try {
    connect_db(uri);
    const server = app.listen(port, () => {
      console.log(`Server listening at http://localhost:${port}`);
    });

    server.on("upgrade", proxyMiddleware.upgrade);
  } catch (error) {
    console.log(error);
  }
};

start();
