export default defineNuxtConfig({
  compatibilityDate: '2024-11-01',
  ssr: false,
  devtools: { enabled: false },
  css: ['~/assets/css/main.css'],
  runtimeConfig: {
    // empty = no backend: the shelf is stored in the visitor's browser (GitHub Pages build)
    public: { apiBase: process.env.NUXT_PUBLIC_API_BASE || '' },
  },
  app: {
    head: {
      title: 'Vinyl Deck — calm focus music',
      meta: [{ name: 'viewport', content: 'width=device-width, initial-scale=1' }],
      link: [
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'stylesheet', href: 'https://fonts.googleapis.com/css2?family=Manrope:wght@400;500;600;700&display=swap' },
      ],
    },
  },
})
