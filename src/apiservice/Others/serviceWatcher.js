const dotenv = require("dotenv");
dotenv.config()
// serviceWatcher.js
class ServiceWatcher {
  static instance; // Singleton instance
  constructor(timeoutDuration = 5000) {
    if (!ServiceWatcher.instance) {
      this.#map = new Map();
      this.#eventsMap = new Map();
      this.#eventsMap.set("onExpire", null);
      this.#timeoutDuration = timeoutDuration; // Configurable timeout
      ServiceWatcher.instance = this;
    }
    return ServiceWatcher.instance;
  }

  // Private fields
  #map;
  #eventsMap;
  #timeoutDuration;

  add(serviceName, currentTime) {
    console.log("Service added for watching: ",serviceName)
    this.#addTimer(serviceName, currentTime);
  }

  delete(serviceName,type = "delete") { // canbe run while update
    if (this.#map.has(serviceName)) {

      clearTimeout(this.#map.get(serviceName));
      this.#map.delete(serviceName);
      if (this.#eventsMap.get("onExpire") && type =="delete") {
        this.#eventsMap.get("onExpire")(serviceName, Date.now());
      }
    }
  }

  update(serviceName, currentTime) {
    this.delete(serviceName,"update");
    this.add(serviceName, currentTime);
  }

  has(serviceName) {
    return this.#map.has(serviceName);
  }

  addEventListener(event, callback) {
    if (typeof event !== "string" || !this.#eventsMap.has(event)) {
      throw new Error("Invalid event");
    }
    if (typeof callback !== "function") {
      throw new Error("Callback must be a function");
    }
    this.#eventsMap.set(event, callback);
  }

  #addTimer(serviceName, currentTime) {
    if (this.#map.has(serviceName)) {
      clearTimeout(this.#map.get(serviceName));
    }
    this.#map.set(
      serviceName,
      setTimeout(() => {
        this.delete(serviceName);
      }, this.#timeoutDuration),
    );
  }
}



const TIMEOUT_MINUTES = process.env.TIMEOUT_MINUTES || 120000
const serviceWatcher = new ServiceWatcher(parseInt(TIMEOUT_MINUTES)); // 10-second timeout

module.exports = {serviceWatcher}
