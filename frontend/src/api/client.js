import axios from 'axios'

const client = axios.create({ baseURL: '/api' })

client.interceptors.request.use((cfg) => {
  const token = sessionStorage.getItem('token')
  if (token) cfg.headers.Authorization = `Bearer ${token}`
  return cfg
})

client.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401 && window.location.pathname !== '/login') {
      sessionStorage.removeItem('token')
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  },
)

export function apiError(err) {
  if (axios.isAxiosError(err)) return err.response?.data?.error || err.message || '请求失败'
  return String(err)
}

export default client
