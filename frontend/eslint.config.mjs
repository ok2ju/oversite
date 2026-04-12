import { defineConfig, globalIgnores } from "eslint/config"
import tseslint from "typescript-eslint"
import reactHooks from "eslint-plugin-react-hooks"
import prettier from "eslint-config-prettier"

const eslintConfig = defineConfig([
  ...tseslint.configs.recommended,
  {
    plugins: { "react-hooks": reactHooks },
    rules: reactHooks.configs.recommended.rules,
  },
  prettier,
  globalIgnores(["dist/**", "build/**", "wailsjs/**"]),
])

export default eslintConfig
