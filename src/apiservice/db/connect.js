import mongoose from "mongoose";
const connect_db =async (uri) => {
  return mongoose.connect(uri).then(() => {
    console.log("DB connected ....");
  });
};
export { connect_db };
