const js = require("@eslint/js");
const ts = require("typescript-eslint");
const vue = require("eslint-plugin-vue");
const vueParser = require("vue-eslint-parser");

module.exports = [
  js.configs.recommended,
  ...ts.configs.recommended,
  ...vue.configs["flat/recommended"],
  {
    files: ["**/*.vue", "**/*.ts", "**/*.tsx"],
    languageOptions: {
      parser: vueParser,
      parserOptions: {
        projectService: true,
        tsconfigRootDir: __dirname,
        extraFileExtensions: [".vue"],
        sourceType: "module",
        ecmaVersion: "latest",
      },
    },
    rules: {
      "vue/multi-word-component-names": "off",
      "vue/no-unused-vars": "error",
      "vue/component-api-style": ["error", ["script-setup", "composition"]],
      "vue/define-macros-order": ["error", {
        order: ["defineProps", "defineEmits", "defineExpose"],
      }],

      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_" }],
      "@typescript-eslint/no-explicit-any": "warn",
      "@typescript-eslint/consistent-type-imports": ["error", { prefer: "type-imports" }],
      "@typescript-eslint/explicit-function-return-type": "off",

      "no-console": ["warn", { allow: ["warn", "error"] }],
      "prefer-const": "error",
    },
  },

  {
    ignores: [
      "dist/**",
      "node_modules/**",
      "*.config.js",
      "*.config.ts",
      "*.config.cjs",
    ],
  },
];
