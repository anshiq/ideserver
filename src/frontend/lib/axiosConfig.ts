import axios from "axios";
const axiosFetch = axios.create({
  baseURL: process.env.backendUrl,
  // baseURL: "http://localhost:8080/user",
  // baseURL: "http://localhost:8080",
  timeout: 8000,
});

const axiosFetchAuth = (token: string) =>
  axios.create({
    baseURL: process.env.backendUrl + "/auth",
    headers: {
      Authorization: `${token}`,
    },
  });
export { axiosFetch, axiosFetchAuth };
