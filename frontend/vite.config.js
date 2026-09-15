import { writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { defineConfig } from 'vite'

// main.go embeds frontend/dist, so the directory has to exist in a fresh clone
// for `go build` and `go test` to work before any frontend build has run. Vite
// wipes the directory on every build, so the placeholder is put back afterwards.
function keepDistTracked() {
  return {
    name: 'keep-dist-tracked',
    closeBundle() {
      writeFileSync(resolve(import.meta.dirname, 'dist/.gitkeep'), '')
    },
  }
}

export default defineConfig({
  plugins: [keepDistTracked()],
})
