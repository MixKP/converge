<script setup lang="ts">
import { ref } from 'vue'
import viteLogo from '../assets/vite.svg'
import heroImg from '../assets/hero.png'
import vueLogo from '../assets/vue.svg'

const count = ref(0)
</script>

<template>
  <section class="flex flex-col items-center justify-center text-center py-16 gap-6">
    <div class="relative flex items-center justify-center">
      <img :src="heroImg" class="w-[170px] h-[179px]" alt="" />
      
      <img
        :src="vueLogo"
        class="absolute w-16 -top-4 -left-6 rotate-[-10deg]"
        alt="Vue logo"
      />
      
      <img
        :src="viteLogo"
        class="absolute w-16 -bottom-4 -right-6 rotate-[10deg]"
        alt="Vite logo"
      />
    </div>
    <div>
      <h1 class="text-4xl font-bold mb-2">Get started</h1>
      <p class="text-gray-500">
        Edit <code class="bg-gray-100 px-1 rounded">src/App.vue</code> and save to test
        <code class="bg-gray-100 px-1 rounded">HMR</code>
      </p>
    </div>
    <button
      type="button"
      class="px-6 py-2 rounded-xl bg-black text-white hover:bg-gray-800 transition"
      @click="count++"
    >
      Count is {{ count }}
    </button>
  </section>

  <div class="w-full h-px bg-gray-200 my-10"></div>

  <section class="grid md:grid-cols-2 gap-8 px-6 max-w-5xl mx-auto">
    <div class="p-6 border rounded-2xl shadow-sm hover:shadow-md transition">
      <h2 class="text-xl font-semibold mb-2">Documentation</h2>
      <p class="text-gray-500 mb-4">Your questions, answered</p>

      <ul class="space-y-3">
        <li>
          <a href="https://vite.dev/" target="_blank" class="flex items-center gap-2 hover:underline">
            <img class="w-5" :src="viteLogo" alt="" />
            Explore Vite
          </a>
        </li>
        <li>
          <a href="https://vuejs.org/" target="_blank" class="flex items-center gap-2 hover:underline">
            <img class="w-5" :src="vueLogo" alt="" />
            Learn more
          </a>
        </li>
      </ul>
    </div>
    <div class="p-6 border rounded-2xl shadow-sm hover:shadow-md transition">
      <h2 class="text-xl font-semibold mb-2">Connect with us</h2>
      <p class="text-gray-500 mb-4">Join the Vite community</p>

      <ul class="space-y-3">
        <li>
          <a href="https://github.com/vitejs/vite" target="_blank" class="hover:underline">
            GitHub
          </a>
        </li>
        <li>
          <a href="https://chat.vite.dev/" target="_blank" class="hover:underline">
            Discord
          </a>
        </li>
        <li>
          <a href="https://x.com/vite_js" target="_blank" class="hover:underline">
            X.com
          </a>
        </li>
        <li>
          <a href="https://bsky.app/profile/vite.dev" target="_blank" class="hover:underline">
            Bluesky
          </a>
        </li>
      </ul>
    </div>
  </section>

  <div class="h-40"></div>
</template>
