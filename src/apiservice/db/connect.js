const mongoose = require("mongoose");
const connect_db =async (uri) => {
  return mongoose.connect(uri)
};
module.exports = { connect_db };
