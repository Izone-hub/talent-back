<script setup>
import { ref } from 'vue'

const count = ref(0)
</script>

<template>
  <div>
    <h1>Hello, Sandbox!</h1>
    <p data-test="count">Count: {{ count }}</p>
    <button data-test="inc" @click="count++">Increment</button>
  </div>
</template>
