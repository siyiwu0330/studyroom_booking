// Swap this mock with your real HTTP client (fetch/axios)
export async function register({ username, password }) {
await new Promise((r) => setTimeout(r, 700))


if (!username || !password) {
const err = new Error('Missing username or password.')
err.status = 400
throw err
}


if (username.toLowerCase() === 'taken') {
const err = new Error('Username already exists.')
err.status = 409
throw err
}


// Example return shape
return { ok: true, userId: Math.floor(Math.random() * 100000) }
}