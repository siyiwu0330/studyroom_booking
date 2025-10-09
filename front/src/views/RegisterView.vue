<!-- src/pages/RegisterPage.vue -->
<template>
  <main class="container">
    <form class="card" @submit.prevent="onSubmit" novalidate>
      <h1>Create your account</h1>

      <!-- Username -->
      <label class="field">
        <span>Username</span>
        <input
          type="text"
          v-model.trim="form.username"
          name="username"
          autocomplete="username"
          required
          :aria-invalid="Boolean(errors.username)"
          @blur="validateUsername"
          placeholder="e.g. alex123"
        />
        <small v-if="errors.username" class="error">{{ errors.username }}</small>
      </label>

      <!-- Password -->
      <label class="field">
        <span>Password</span>
        <div class="password-wrap">
          <input
            :type="showPassword ? 'text' : 'password'"
            v-model="form.password"
            name="password"
            autocomplete="new-password"
            required
            :aria-invalid="Boolean(errors.password)"
            @blur="validatePassword"
            placeholder="Min 8 characters"
          />
          <button
            type="button"
            class="toggle"
            @click="showPassword = !showPassword"
            :aria-pressed="showPassword.toString()"
          >
            {{ showPassword ? 'Hide' : 'Show' }}
          </button>
        </div>
        <small v-if="errors.password" class="error">{{ errors.password }}</small>
        <ul class="hints">
          <li :class="{ ok: passLenOK }">At least 8 characters</li>
          <li :class="{ ok: passMixOK }">Letters & numbers</li>
        </ul>
      </label>

      <button class="submit" type="submit" :disabled="!isFormValid || loading">
        {{ loading ? 'Creating...' : 'Register' }}
      </button>

      <p v-if="serverMsg" class="server" :class="{ ok: serverOk }">{{ serverMsg }}</p>
    </form>
  </main>
</template>

<script setup>
import { reactive, computed, ref } from 'vue'

const form = reactive({ username: '', password: '' })
const errors = reactive({ username: '', password: '' })

const showPassword = ref(false)
const loading = ref(false)
const serverMsg = ref('')
const serverOk = ref(false)

const passLenOK = computed(() => form.password.length >= 8)
const passMixOK = computed(() => /[A-Za-z]/.test(form.password) && /\d/.test(form.password))
const isFormValid = computed(() => form.username && passLenOK.value && passMixOK.value)

function validateUsername() {
  errors.username = ''
  if (!form.username) errors.username = 'Username is required.'
  else if (form.username.length < 3) errors.username = 'Username must be at least 3 characters.'
  else if (!/^[a-zA-Z0-9._-]+$/.test(form.username)) errors.username = 'Only letters, numbers, dot, underscore, and hyphen are allowed.'
}

function validatePassword() {
  errors.password = ''
  if (!form.password) errors.password = 'Password is required.'
  else if (!passLenOK.value) errors.password = 'Password must be at least 8 characters.'
  else if (!passMixOK.value) errors.password = 'Use letters and numbers.'
}

async function onSubmit() {
  validateUsername(); validatePassword()
  if (errors.username || errors.password) return

  loading.value = true; serverMsg.value = ''; serverOk.value = false
  try {
    // Replace with your real API call:
    // const resp = await fetch('/api/register', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(form) })
    // if (!resp.ok) throw new Error((await resp.json()).message || 'Registration failed')

    await new Promise(r => setTimeout(r, 600)) // demo delay
    if (['admin','root'].includes(form.username.toLowerCase())) {
      throw new Error('That username is not available.')
    }
    serverOk.value = true
    serverMsg.value = 'Account created! You can sign in now.'
    // optional: navigate with router.push('/login')
  } catch (e) {
    serverOk.value = false
    serverMsg.value = e.message || 'Something went wrong.'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
:root { color-scheme: light dark; }
.container {
  min-height: 100dvh; display: grid; place-items: center; padding: 2rem;
  background: var(--bg, Canvas);
}
.card {
  width: 100%; max-width: 420px; padding: 1.5rem; border-radius: 12px;
  background: var(--card, Canvas); box-shadow: 0 6px 24px rgba(0,0,0,.08);
}
h1 { margin: 0 0 1rem; font-size: 1.4rem; }
.field { display: block; margin: 0 0 1rem; }
.field > span { display: inline-block; margin-bottom: .4rem; font-weight: 600; }
input {
  width: 100%; padding: .7rem .9rem; border: 1px solid var(--bd, #d0d7de);
  border-radius: 10px; font: inherit; outline: none;
}
input[aria-invalid="true"] { border-color: #e5484d; }
.password-wrap { position: relative; }
.toggle {
  position: absolute; right: .4rem; top: 50%; transform: translateY(-50%);
  border: none; background: transparent; padding: .35rem .5rem; cursor: pointer; font: inherit;
}
.error { color: #e5484d; display: block; margin-top: .35rem; }
.hints { margin: .4rem 0 0; padding-left: 1.1rem; font-size: .9rem; color: #6b7280; }
.hints li { margin: .15rem 0; list-style: disc; }
.hints li.ok { color: #16a34a; }
.submit {
  width: 100%; padding: .8rem 1rem; border: 0; border-radius: 10px;
  background: #3b82f6; color: #fff; font-weight: 700; cursor: pointer;
}
.submit:disabled { opacity: .6; cursor: not-allowed; }
.server { margin-top: .8rem; }
.server.ok { color: #16a34a; }
@media (prefers-color-scheme: dark) {
  :root { --bg: #0b0c0f; --card: #111318; --bd: #2a2f3a; }
}
</style>
