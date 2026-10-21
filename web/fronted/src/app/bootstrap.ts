import {createApp} from 'vue'
import App from '@/App.vue'
import router from '@/router/index'
import '@/assets/global.css'
import '@/assets/toast.css'
import '@/assets/modal.css'
import '@/assets/workspace.css'
// 图标字体本地打包：内网部署无法访问 cdnjs，CDN 不可达会导致全站图标消失。
import '@fortawesome/fontawesome-free/css/all.min.css'
import {initMobileOptimizations} from '@/utils/mobile-utils'
import {initSessionExpiration} from '@/app/session-expiration.service'

export function bootstrapApp(): void {
    createApp(App).use(router).mount('#app')

    initMobileOptimizations()
    initSessionExpiration()

    console.info('[main] 应用入口加载时间:', new Date().toISOString())
    window.__APP_MAIN_LOADED = true
}

