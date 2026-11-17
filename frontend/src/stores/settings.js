import { defineStore } from 'pinia'

export const useSettingsStore = defineStore('settings', {
  state: () => ({
    language: 'id',
    theme: 'light',
    fullscreen: false
  }),
  getters: {
    isDark: (s) => s.theme === 'dark',
    isFullscreen: (s) => s.fullscreen,
    langLabel: (s) => (s.language === 'id' ? 'ID' : 'EN')
  },
  actions: {
    setLanguage(lang) {
      this.language = lang
      localStorage.setItem('lang', lang)
    },
    setTheme(theme) {
      this.theme = theme
      localStorage.setItem('theme', theme)
      document.documentElement.classList.toggle('dark', theme === 'dark')
    },
    toggleTheme() {
      this.setTheme(this.theme === 'dark' ? 'light' : 'dark')
    },
    toggleLanguage() {
      this.setLanguage(this.language === 'id' ? 'en' : 'id')
    },
    toggleFullscreen() {
      this.fullscreen = !this.fullscreen
      localStorage.setItem('fullscreen', this.fullscreen)
    }
  }
})