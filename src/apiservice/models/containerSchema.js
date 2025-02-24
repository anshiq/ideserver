const mongoose = require("mongoose");

const containerSchema = new mongoose.Schema({
    name: {
        type: String,
        required: true
    },
    stack: {
        type: String,
        required: true
    },
    yamlCode: {
        type: String,
        required: true
    },
    iconUrl: {
        type: String,
        required: true
    }
});

const Container = mongoose.model("containers", containerSchema);

const generateCustomId = () => {
    const prefix = String.fromCharCode(97 + Math.floor(Math.random() * 26)); // Random lowercase letter (a-z)
    const suffix = new mongoose.Types.ObjectId().toString().substring(1); // Get the rest of the ObjectId
    return prefix + suffix; // Ensure it always starts with a lowercase letter
  };
  
  const serviceSchema = new mongoose.Schema({
    _id: {
      type: String,
      unique: true,
      required: true,
      default: generateCustomId, // Use custom function for _id
      validate: {
        validator: function (v) {
          return /^[a-z][a-z0-9]+$/.test(v); // Ensure starts with lowercase and contains only lowercase & numbers
        },
        message: "ID must start with a lowercase letter and contain only lowercase letters and numbers.",
      },
    },
    name: {
      type: String,
      required: true,
    },
    linkedContainer: {
      type: mongoose.Schema.Types.ObjectId,
      ref: "containers",
    },
    status: {
      type: String,
      enum: ["active", "pending", "terminated", "spawning"],
      required: true,
    },
    userId: {
      type: mongoose.Schema.Types.ObjectId,
      ref: "users",
    },
  });
  
  // Ensure `_id` follows the format before saving
  serviceSchema.pre("validate", function (next) {
    if (!this._id || !/^[a-z][a-z0-9]+$/.test(this._id)) {
      this._id = generateCustomId();
    }
    next();
  });
  
  const Service = mongoose.model("services", serviceSchema);

module.exports = { Container, Service };
