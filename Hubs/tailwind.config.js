/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        hub: {
          dark: '#0a0a0f',
          darker: '#050508',
          border: '#1a1a24',
          surface: '#12121a',
          accent: '#0a84ff',
        },
        civilian: '#30d158',
        military: '#ff9f0a',
        interstellar: '#bf5af2',
      },
      fontFamily: {
        sans: ['-apple-system', 'BlinkMacSystemFont', 'SF Pro Display', 'sans-serif'],
        mono: ['SF Mono', 'Menlo', 'monospace'],
      },
    },
  },
  plugins: [],
};
