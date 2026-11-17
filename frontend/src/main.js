import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { useSettingsStore } from './stores/settings'
import './style.css'
import './dark.css'

const app = createApp(App)
app.use(createPinia())

const settings = useSettingsStore()
const savedLang = localStorage.getItem('lang')
const savedTheme = localStorage.getItem('theme')
if (savedLang) settings.language = savedLang
if (savedTheme) settings.theme = savedTheme
settings.setTheme(savedTheme || 'light')

app.use(router)
app.mount('#app')
