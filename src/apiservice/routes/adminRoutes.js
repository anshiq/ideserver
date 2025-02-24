const { Router } = require("express");
const { createContainer, updateContainer, deleteContainer, getContainer, getAllContainers } = require("../controller/adminController");

const adminRoutes = Router()
adminRoutes.route("/container/:id").patch(updateContainer).delete(deleteContainer).get(getContainer)
adminRoutes.route("/container").post(createContainer).get(getAllContainers)

module.exports = {adminRoutes}