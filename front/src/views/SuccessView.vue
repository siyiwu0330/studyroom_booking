<template>
<div class="card center">
<h2>Account created 🎉</h2>
<p>You can now sign in.</p>
<router-link to="/register" class="link">Back to register</router-link>
</div>
</template>


<script setup>
</script>


<style scoped>
.card { width: 360px; max-width: 92vw; background:white; padding: 24px; border-radius: 14px; box-shadow: 0 10px 25px rgba(0,0,0,.08); text-align:center; }
.link { display:inline-block; margin-top: 10px; text-decoration: underline; }
</style>