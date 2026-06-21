import { reactRouter } from "@react-router/dev/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig, type PluginOption } from "vite";
import tsconfigPaths from "vite-tsconfig-paths";

export default defineConfig(({ mode }) => ({
  plugins: [tailwindcss(), mode === "test" ? undefined : reactRouter(), tsconfigPaths()].filter(
    Boolean,
  ) as PluginOption[],
  server: {
    host: "127.0.0.1",
    port: 3002,
  },
  test: {
    environment: "jsdom",
    exclude: ["tests/e2e/**", "node_modules/**", "build/**", ".react-router/**"],
    globals: true,
    setupFiles: ["./tests/setup.ts"],
  },
}));
