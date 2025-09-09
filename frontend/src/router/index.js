import { createRouter, createWebHashHistory } from 'vue-router'
import ContactsScreen from '../components/views/ContactsScreen.vue'
import ChatWindow from '../components/views/ChatWindow.vue'
import ConnectScreen from '../components/views/ConnectScreen.vue'
import SettingsScreen from '../components/views/SettingsScreen.vue'

const routes = [
  {
    path: '/',
    name: 'Connect',
    component: ConnectScreen
  },
  {
    path: '/contacts',
    name: 'Contacts',
    component: ContactsScreen
  },
  {
    path: '/chat/:jid',
    name: 'Chat',
    component: ChatWindow,
    props: true
  },
  {
    path: '/settings',
    name: 'Settings',
    component: SettingsScreen
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

export default router
