/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./src/**/*.{js,jsx,ts,tsx}', './public/index.html', "./node_modules/tailwind-datepicker-react/dist/**/*.js",],
  theme: {
    extend: {
      colors: {
      },
      fontWeight: {
        'extra-bold': 1800,
      },
      fontFamily: {
        sans: ['Inter', 'sans-serif'],
      },
    },
  },
  plugins: [require("@tailwindcss/forms")],
  darkMode: 'selector',
};
