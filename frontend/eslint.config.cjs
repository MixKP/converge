const js = require('@eslint/js')
const ts = require('typescript-eslint')
const vue = require('eslint-plugin-vue')
const vueParser = require('vue-eslint-parser')
const prettier = require('eslint-config-prettier')
const prettierPlugin = require('eslint-plugin-prettier')
const tailwind = require('eslint-plugin-tailwindcss')

module.exports = [
  js.configs.recommended,
  ...ts.configs.recommended,
  ...vue.configs['flat/recommended'],
  tailwind.configs.recommended,

  {
    settings: {
      tailwindcss: {
        cssConfigPath: 'src/css/style.css',
      },
    },
  },

  {
    files: ['**/*.vue', '**/*.ts', '**/*.tsx'],
    plugins: {
      prettier: prettierPlugin,
    },
    languageOptions: {
      parser: vueParser,
      parserOptions: {
        parser: ts.parser,
        projectService: true,
        tsconfigRootDir: __dirname,
        extraFileExtensions: ['.vue'],
        sourceType: 'module',
        ecmaVersion: 'latest',
      },
      globals: {
        window: 'readonly',
        setTimeout: 'readonly',
        clearTimeout: 'readonly',
        setInterval: 'readonly',
        clearInterval: 'readonly',
      },
    },
    rules: {
      'prettier/prettier': 'error',

      'vue/multi-word-component-names': 'off',
      'vue/no-unused-vars': 'error',
      'vue/component-api-style': ['error', ['script-setup', 'composition']],

      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      '@typescript-eslint/no-explicit-any': 'warn',
      '@typescript-eslint/consistent-type-imports': ['error', { prefer: 'type-imports' }],

      'no-console': ['warn', { allow: ['warn', 'error'] }],
      'prefer-const': 'error',
    },
  },

  prettier,

  {
    ignores: ['dist/**', 'node_modules/**', '*.config.js', '*.config.ts', '*.config.cjs'],
  },
]
