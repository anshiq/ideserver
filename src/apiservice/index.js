import express, { Express } from "express";
import dotenv from "dotenv";
import cors from "cors";
import { userRouter } from "./routes/userRoutes";
import { connect_db } from "./db/connect";
import { verifyToken } from "./middlewares/authMiddleware";
import { authRouter } from "./routes/authRoutes";
dotenv.config();
const port = process.env.PORT || 8080;
const uri = process.env.MONGOURI || "";
const app = express();
app.use(express.json());
app.use(cors());
app.use("/user", userRouter);
app.use("/auth", verifyToken, authRouter);
const start = () => {
  try {
    connect_db(uri);
    app.listen(port, () => {
      console.log(`Server listening at http://localhost:${port}`);
    });
  } catch (error) {
    console.log(error);
  }
};
start();
