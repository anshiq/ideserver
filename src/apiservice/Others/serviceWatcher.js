const dotenv = require("dotenv")
dotenv.config()
class ServiceWatcher {
  static instance; // Singleton instance
  constructor(timeoutDuration = 5000) {
    if (!ServiceWatcher.instance) {
      this.#map = new Map();
      this.#eventsMap = new Map();
      this.#eventsMap.set("onChange", null);
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
    // Call the private method to add a timer
    this.#addTimer(serviceName, currentTime);
    
    // Trigger onChange event if it exists
    if (this.#eventsMap.get("onChange")) {
      this.#eventsMap.get("onChange")(serviceName, currentTime);
    }
  }
  
  delete(serviceName) {
    if (this.#map.has(serviceName)) {
      clearTimeout(this.#map.get(serviceName));
      this.#map.delete(serviceName);
    }
  }
  
  update(serviceName, currentTime) {
    this.delete(serviceName);
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
  
  // Private method to manage timers
  #addTimer(serviceName, currentTime) {
    if (this.#map.has(serviceName)) {
      clearTimeout(this.#map.get(serviceName));
    }
    this.#map.set(
      serviceName,
      setTimeout(() => {
        this.delete(serviceName);
      }, this.#timeoutDuration)
    );
  }
}

// Usage
const TIMEOUT_MINUTES = process.env.TIMEOUT_MINUTES || 120000
const serviceWatcher = new ServiceWatcher(parseInt(TIMEOUT_MINUTES)); // 10-second timeout
module.exports = {serviceWatcher}
