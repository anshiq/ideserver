import { Router } from "express";
import { getUserDetails } from "../controller/authedController";
const authRouter = Router();
authRouter.route("/api").get(getUserDetails);
export { authRouter };
