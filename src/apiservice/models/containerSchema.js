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

const serviceSchema = new mongoose.Schema({
    name: {
        type: String,
        required: true
    },
    linkedContainer: {
        type: mongoose.Schema.Types.ObjectId,
        ref: "containers"
    },
    status: {
        type: String,
        enum: ["active", "pending", "terminated","spawning"],
        required: true
    }, 
    userId: {
        type: mongoose.Schema.Types.ObjectId,
        ref: "users"
    }
});

const Service = mongoose.model("services", serviceSchema);

module.exports = { Container, Service };
