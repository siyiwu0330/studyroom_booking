<script setup>
import { reactive, ref, computed } from 'vue'
const emit = defineEmits(['submit'])

const form = reactive({ username: '', password: '' })
const touched = reactive({ username: false, password: false })
const showPassword = ref(false)
const submitting = ref(false)

const usernameError = computed(() => {
  if (!form.username) return 'Username is required.'
  if (form.username.length < 3) return 'Username must be at least 3 characters.'
  if (!/^[a-zA-Z0-9._-]+$/.test(form.username)) return 'Use letters, numbers, dot, underscore, or dash.'
  return ''
})

const passwordError = computed(() => {
  if (!form.password) return 'Password is required.'
  if (form.password.length < 8) return 'Password must be at least 8 characters.'
  return ''
})

const formValid = computed(() => !usernameError.value && !passwordError.value)

async function onSubmit() {
  touched.username = true
  touched.password = true
  if (!formValid.value) return
  submitting.value = true
  try {
    await new Promise(r => setTimeout(r, 50)) // allow button state update
    emit('submit', { ...form })
  } finally {
    submitting.value = false
  }
}
</script>
