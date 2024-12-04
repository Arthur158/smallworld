import axios, { AxiosInstance, InternalAxiosRequestConfig } from 'axios';
import { API_URL } from '../constants';

const httpService: AxiosInstance = axios.create({
  baseURL: API_URL,
});

export const attachToken = (request: InternalAxiosRequestConfig): InternalAxiosRequestConfig => {

  const token = localStorage.getItem('token');
  if (token) request.headers.Authorization = `Bearer ${token}`

  return request;
};

httpService.interceptors.request.use(attachToken);

export default httpService;
