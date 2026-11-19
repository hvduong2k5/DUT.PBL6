import axios from "axios";

export const apiClient = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL || "http://localhost:8016/api/v1",
  withCredentials: true,
  headers: {
    "Content-Type": "application/json"
  }
});
