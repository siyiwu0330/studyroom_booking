<template>
  <header class="nav">
    <RouterLink to="/">Home</RouterLink>
    <RouterLink to="/register">Register</RouterLink>
    <RouterLink to="/login">Login</RouterLink>
  </header>

  <RouterView />
</template>

<script setup>
// nothing needed here
</script>

<style scoped>
.nav { display:flex; gap:12px; padding:12px; border-bottom:1px solid #e5e7eb; }
a.router-link-active { font-weight:700; }
</style>
