const { Container } = require("../models/containerSchema")


const createContainer = async (req, res) => {
    try {
        const { name, yamlCode, stack, iconUrl } = req.body;
        if (!name || !yamlCode || !stack) {
            return res.status(400).json({ error: "Missing required fields" });
        }
        const container = await Container.create({ name, yamlCode, stack, iconUrl });
        res.status(201).json(container);
    } catch (error) {
        res.status(500).json({ error: "Server error", details: error.message });
    }
};

const getContainer = async (req, res) => {
    try {
        const { id } = req.params;
        const container = await Container.findById(id);
        if (!container) {
            return res.status(404).json({ error: "Container not found" });
        }
        res.json(container);
    } catch (error) {
        res.status(500).json({ error: "Server error", details: error.message });
    }
};

const getAllContainers = async (req, res) => {
    try {
        const containers = await Container.find();
        res.json(containers);
    } catch (error) {
        res.status(500).json({ error: "Server error", details: error.message });
    }
};

const updateContainer = async (req, res) => {
    try {
        const { id } = req.params;
        const { name, yamlCode, stack, iconUrl } = req.body;
        const updatedContainer = await Container.findByIdAndUpdate(id, { name, yamlCode, stack, iconUrl }, { new: true });
        if (!updatedContainer) {
            return res.status(404).json({ error: "Container not found" });
        }
        res.json(updatedContainer);
    } catch (error) {
        res.status(500).json({ error: "Server error", details: error.message });
    }
};

const deleteContainer = async (req, res) => {
    try {
        const { id } = req.params;
        const deletedContainer = await Container.findByIdAndDelete(id);
        if (!deletedContainer) {
            return res.status(404).json({ error: "Container not found" });
        }
        res.json({ message: "Container deleted successfully" });
    } catch (error) {
        res.status(500).json({ error: "Server error", details: error.message });
    }
};

module.exports = { createContainer, getContainer, getAllContainers, updateContainer, deleteContainer };
