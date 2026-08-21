import { ref } from 'vue'
import client from './api/client'

const legacyToken = localStorage.getItem('token') || ''
if (legacyToken && !sessionStorage.getItem('token')) {
  sessionStorage.setItem('token', legacyToken)
}
localStorage.removeItem('token')

const token = ref(sessionStorage.getItem('token') || '')
const username = ref('')
const ready = ref(false)

async function login(u, p) {
  const res = await client.post('/auth/login', { username: u, password: p })
  token.value = res.data.token
  username.value = res.data.username
  sessionStorage.setItem('token', res.data.token)
}

function logout() {
  token.value = ''
  username.value = ''
  sessionStorage.removeItem('token')
  localStorage.removeItem('token')
}

async function loadMe() {
  if (!token.value) {
    ready.value = true
    return
  }
  try {
    const res = await client.get('/auth/me')
    username.value = res.data.username
  } catch {
    logout()
  } finally {
    ready.value = true
  }
}

export function useAuth() {
  return { token, username, ready, login, logout, loadMe }
}
