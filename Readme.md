Building codesandbox or replit for myself 🙃. 
overview: 
runner service: check if its running in cluster and has rbac permissions. or looks for kube config file. to spin deployments each deployment is connected to environment spinned by user and has control over code server via ingress proxy.
apiservice : handles record of spinned deployments. their configs ,users and timeout logic for not used deployments (user environments) after a long time frame.

these both communicate via grpc.

front-end: portal for user to use the project least developed.
*detailed explanation pending*
