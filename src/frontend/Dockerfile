# Use Node 21 Alpine as the base image
FROM node:21-alpine
WORKDIR /code
COPY . .
ENV BACKEND_URL="https://api.iamanshik.online"
RUN npm install && npm run build
CMD ["npm", "start"]
