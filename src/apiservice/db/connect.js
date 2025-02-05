const mongoose = require("mongoose");
const connect_db =async (uri) => {
  return mongoose.connect(uri).then(() => {
    console.log("DB connected ....");
  });
};
module.exports = { connect_db };
