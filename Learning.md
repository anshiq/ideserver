# gRPC

### Generating gRPC Code from Protobuf
- Used `genproto.sh` script to generate gRPC code from `.proto` files.
- Note: Service names and methods defined in the `.proto` file cannot be redefined or linked directly to a struct in the Go/JS code. Instead, the generated gRPC code provides interfaces and methods to implement or use these services.


# Kubernetes (K8S)

### Cluster
- A **Kubernetes Cluster** is a network of machines, consisting of:
  - **Control Plane (Head Node):** Manages the cluster and makes decisions like scheduling containers.
  - **Worker Nodes:** Execute the workloads (containers).
- The combination of the control plane and worker nodes makes up the Kubernetes cluster.

### Containers
- **Container Images:** 
  - Created using **Docker** or similar tools.
  - Stored in repositories like **Docker Hub** or private registries.
  - Pulled into Kubernetes for deployment.
  
- **YAML Files for Deployment:**
  - **`deployment.yaml`:** Describes how to deploy and manage the containerized application.
  - **`service.yaml`:** Defines the networking setup (e.g., exposing containers via a load balancer or cluster IP).

- **In-Cluster User Account:**
  - Kubernetes **automatically mounts a user account token** in all containers.
  - This token grants containers full control of the cluster unless restricted by Role-Based Access Control (RBAC).
  - k8s inCluser User account is mounted to all these containers by default, means each container can have full controll the k8s cluster if it wants.
#### Key Fields in `deployment.yaml`:
1. **`metadata`:**
   - `name`: Unique name for each deployment or service.
     - Used for identifying, deleting, or updating resources.
   - `namespace`: Groups resources logically (e.g., `production`, `staging`, or `development`).

2. **`spec`:**
   - Defines the configuration for deployments, such as:
     - Number of replicas (`replicas` field).
     - Selector (`selector.matchLabels`) to target specific Pods (e.g., all database Pods).

3. **`spec.template.spec`:**
   - Configures container properties, including:
     - Container name.
     - Image to use.
     - Ports to expose.
     - Account or credentials to mount.

---

### Pods
- **Definition:** A Pod is the smallest deployable unit in Kubernetes, consisting of one or more containers that:
  - Share the same **network namespace** (IP address, port space).
  - Share the same **storage** (e.g., persistent volumes).
- **Usage:** Pods are created for tightly coupled containers. 
  - Example: Multiple PostgreSQL containers in a Pod sharing the same volume for storing data for another kind of container even if it is single, a new pod for that will be generated.

---

### Volumes
1. Persistant Volume: A complete storage which can be access by pod independent of pod behaviour like storage for sql database containers containing pod.
2. Non persistant Volume (afemeral volumes): Lets assume two pods need to share some data through storage, after sharing if both pods gets destroyed the storage volume will also be destroyed.
Provisioning_Types: i) Static Provisioning ii) Dynamic Provisioning
**Static_Provisioning:** We specify no of gigs in cluster as static even before any pod provisioning or any thing.
- Lets say we provision 3 static each 5gb volume with permisions of (read write many,ready only many,read write single)
- Now if pod have to access volume in cluster, the cluster will  automatically assign the volume to pod for its given set of access mode(rwm) and storage amount, if storage is not enough or no volume statisfy pod access mode it will reject request.
- If condition statisfy, lets say the pod required 3gb as storage, now k8s will decrease that selected volume to 2gb from 5gb, cause 3gb is assigned to pod.




---

# Docker
### Commands in docker file
FROM is used to select a default existing image to start from.
USER root to select root as user in docker file.
RUN is used to execute commands in image before building image from dockerfile
eg: RUN useradd -m -u 1001 -d /home/vscode -s /bin/bash vscode
RUN echo "vscode ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers
is command used to create a user vscode in docker image, and give all sudoer permissions
ENTRYPOINT is last point of image at which we define what proccess this container used for to execute.

if we mount a volume to container just before ENTRYPOINT the volume is mounted,

lets say we mount /host:/home/image/code in container, before mounting if code folder already exists in image code, after mounting it is overwritten by the folder present in dist of host,
now lets say we have to transfer some files to this mounted folder check docker file in supportedImages folder, we create transfer these imp files to other folder while creating image, and when we create image to container using a mount we add a script at ENTRYPOINT which does all file transfer be running main executable like code-server
some times host folder don't give permission to write then on the host terminal we can chmod +777 on that mounted folder.


 

# Golang
*(To be filled)*

---

# JavaScript
*(To be filled)*
