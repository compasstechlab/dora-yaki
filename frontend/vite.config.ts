import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    port: 7201,
    proxy: {
      "/api": {
        target: "http://localhost:7202",
        changeOrigin: true,
      },
    },
  },
});
