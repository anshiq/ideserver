
import { User } from "../models/userSchema";
const getUserDetails = async (req, res) => {
  try {
    const data = await User.findById(req.userId);
    if (!data) return;
    const user = {
      success: true,
      data: {
        name: data.name,
        mobile: data.mobile,
        email: data.email,
      },
    };
    res.json(user);
  } catch (error) {
    res.json({ success: false, data: { msg: JSON.stringify({ err: error }) } });
  }
};
export { getUserDetails };
