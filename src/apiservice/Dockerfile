# Use Node 21 Alpine as the base image
FROM node:21-alpine

# Set the working directory
WORKDIR /code

# Copy package.json and package-lock.json first to leverage Docker cache
COPY package*.json ./

# Install dependencies
# RUN apk add  make gcc g++ python3
# to install bcrypt
RUN npm install 
# RUN npm rebuild bcrypt -–build-from-source
# breaking in environment of k8s so building it natively so building it natively
# RUN apk del make gcc g++ python3

# Copy the rest of the application code
COPY . .

# Set environment variables
# ENV PORT=8080 \
#     GRPC_PORT="localhost:8081" \
#     MONGOURI="mongodb+srv://anshik:anshik12@cluster0.fuqim.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0" \
#     EMAIL_SERVICE_MAIL="anshikthind@gmail.com" \
#     EMAIL_SERVICE_PASS="bqmj ktff rpuy dakx" \
#     SALTROUNDS=12 \
#     JWTSECRET="zehaha" \
#     FRONTEND_URL="https://iamanshik.online" \
#     API_SERVICE_PUBLIC_IP="https://api.iamanshik.online"

# Expose the port the app runs on
# EXPOSE $PORT

# Command to run the application
# CMD ["npm", "run","start"] # will be handled by yaml deployement
# testing
