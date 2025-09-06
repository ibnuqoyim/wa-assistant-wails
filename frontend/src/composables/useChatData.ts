import { computed, reactive, ref, onMounted } from 'vue'

// Import Wails runtime for events
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { CheckWhatsAppConnection, ConnectExistingDevice, StartNewConnection, RequestPairingCode, GetConnectionStatus, GetChats, GetMessages } from '../../wailsjs/go/main/App'
import { whatsapp } from '../../wailsjs/go/models'


// ---- STATE KONEKSI ----
const isLinked = ref(false) // false = belum konek; true = sudah konek
const qrSrc = ref('') // bisa diisi data URL dari backend Go (WhatsMeow)
const connectionStatus = ref('checking') // 'checking', 'disconnected', 'connecting', 'connected'
const pairingCode = ref('')
const phoneNumber = ref('')

// Setup event listeners immediately (not in onMounted)
console.log('Setting up WhatsApp event listeners...')

// Listen for QR code events
EventsOn('whatsapp:qr', (qrCode: string) => {
  console.log('✅ Received QR code event:', qrCode)
  console.log('QR code length:', qrCode.length)
  console.log('Setting connectionStatus to disconnected')
  connectionStatus.value = 'disconnected'
  qrSrc.value = qrCode
  console.log('qrSrc.value set to:', qrSrc.value.substring(0, 50) + '...')
  // Change status so QR code is displayed
})

// Listen for connection events
EventsOn('whatsapp:connected', (message: string) => {
  console.log('✅ WhatsApp connected:', message)
  connectionStatus.value = 'connected'
  isLinked.value = true
  qrSrc.value = ''
  pairingCode.value = ''
  // Load chats when connected
  loadChats()
})

EventsOn('whatsapp:disconnected', (message: string) => {
  console.log('❌ WhatsApp disconnected:', message)
  connectionStatus.value = 'disconnected'
  isLinked.value = false
  qrSrc.value = ''
  pairingCode.value = ''
  // Clear chats when disconnected
  chats.value = []
  Object.keys(messagesByChat).forEach(key => {
    delete messagesByChat[Number(key)]
  })
})

EventsOn('whatsapp:error', (message: string) => {
  console.error('❌ WhatsApp error:', message)
  connectionStatus.value = 'disconnected'
})

console.log('✅ WhatsApp event listeners setup complete')

// Listen for test startup event
EventsOn('app:startup', (message: string) => {
  console.log('🚀 Test event received:', message)
})

// Setup other initialization in onMounted
onMounted(() => {
  console.log('Component mounted - event listeners already setup')
})

async function checkExistingConnection() {
  try {
    connectionStatus.value = 'checking'
    const status = await CheckWhatsAppConnection()
    
    if (status.isConnected) {
      // Try to reconnect existing device
      await ConnectExistingDevice()
      isLinked.value = true
      connectionStatus.value = 'connected'
      console.log('Reconnected to existing WhatsApp device')
      // Load chats when reconnected
      loadChats()
    } else {
      connectionStatus.value = 'disconnected'
      console.log('No existing WhatsApp connection found')
    }
  } catch (error) {
    console.error('Error checking WhatsApp connection:', error)
    connectionStatus.value = 'disconnected'
  }
}

async function linkWithQR() {
  try {
    console.log('🔄 Starting linkWithQR...')
    connectionStatus.value = 'connecting'
    qrSrc.value = '' // Clear previous QR code
    console.log('📞 Calling StartNewConnection...')
    await StartNewConnection()
    console.log('✅ StartNewConnection completed - waiting for QR event...')
    // QR code should be received via events
  } catch (error) {
    console.error('❌ Error starting QR connection:', error)
    connectionStatus.value = 'disconnected'
  }
}

async function linkWithPhone() {
  const phone = prompt('Enter your phone number (with country code, e.g., +1234567890):')
  if (!phone) return
  
  try {
    connectionStatus.value = 'connecting'
    phoneNumber.value = phone
    const code = await RequestPairingCode(phone)
    pairingCode.value = code
    alert(`Your pairing code is: ${code}`)
  } catch (error) {
    console.error('Error requesting pairing code:', error)
    connectionStatus.value = 'disconnected'
    alert('Failed to request pairing code')
  }
}

function disconnect() {
  isLinked.value = false
  connectionStatus.value = 'disconnected'
  qrSrc.value = ''
  pairingCode.value = ''
}


// ---- DATA CHAT FROM BACKEND ----
interface FrontendChat {
  id: number
  chatId: string
  name: string
  last: string
  time: string
  unread: number
  isGroup: boolean
}

interface FrontendMessage {
  id: number
  author: string
  text: string
  time: string
  mine: boolean
  type?: string
}

const chats = ref<FrontendChat[]>([])
const messagesByChat = reactive<Record<number, FrontendMessage[]>>({})

// Load chats from backend
async function loadChats() {
  try {
    if (!isLinked.value) {
      // Use fallback data when not connected
      chats.value = [
        { id: 1, chatId: '', name: 'Connect WhatsApp', last: 'Please connect to WhatsApp first', time: '', unread: 0, isGroup: false }
      ]
      return
    }

    console.log('Loading chats from backend...')
    const backendChats = await GetChats()
    
    // Transform backend data to frontend format
    chats.value = backendChats.map((chat, index) => ({
      id: index + 1, // Use numeric ID for frontend compatibility
      chatId: chat.id, // Store original WhatsApp ID
      name: chat.name,
      last: chat.last,
      time: chat.time,
      unread: chat.unread,
      isGroup: chat.isGroup
    }))
    
    console.log('Loaded chats:', chats.value)
  } catch (error) {
    console.error('Error loading chats:', error)
    // Fallback to demo data
    chats.value = [
      { id: 1, chatId: 'demo1', name: 'John Doe', last: 'Hello, how are you?', time: '10:30', unread: 2, isGroup: false },
      { id: 2, chatId: 'demo2', name: 'Family Group', last: 'See you tomorrow!', time: '09:15', unread: 0, isGroup: true }
    ]
  }
}

// Load messages for a specific chat
async function loadMessages(chatId) {
  try {
    if (!isLinked.value) return
    
    const chat = chats.value.find(c => c.id === chatId)
    if (!chat || !chat.chatId) return
    
    console.log('Loading messages for chat:', chat.chatId)
    const backendMessages = await GetMessages(chat.chatId, 50)
    
    // Transform backend data to frontend format
    const messages = backendMessages.map((msg, index) => ({
      id: index + 1,
      author: msg.author,
      text: msg.text,
      time: msg.time,
      mine: msg.mine,
      type: msg.type
    }))
    
    messagesByChat[chatId] = messages
    console.log('Loaded messages for chat', chatId, ':', messages)
  } catch (error) {
    console.error('Error loading messages:', error)
    // Fallback to demo messages
    messagesByChat[chatId] = [
      { id: 1, author: 'Demo', text: 'Please connect to WhatsApp to see real messages', time: '00:00', mine: false }
    ]
  }
}


const selectedId = ref(1)
const q = ref('')
const draft = ref('')
const showInfo = ref(true)


const activeChat = computed(() => chats.value.find(c => c.id === selectedId.value))
const activeMessages = computed(() => messagesByChat[selectedId.value] || [])
const totalUnread = computed(() => chats.value.reduce((sum, chat) => sum + chat.unread, 0))

const filteredChats = computed(() => {
  if (!q.value) return chats.value
  return chats.value.filter(chat => 
    chat.name.toLowerCase().includes(q.value.toLowerCase()) ||
    chat.last.toLowerCase().includes(q.value.toLowerCase())
  )
})

function selectChat(id: number) {
  selectedId.value = id
  draft.value = ''
  // Load messages for the selected chat
  loadMessages(id)
}

function sendMessage() {
  if (!draft.value.trim()) return
  
  const newMessage: FrontendMessage = {
    id: Date.now(),
    author: 'Me',
    text: draft.value,
    time: new Date().toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' }),
    mine: true
  }
  
  if (!messagesByChat[selectedId.value]) {
    messagesByChat[selectedId.value] = []
  }
  messagesByChat[selectedId.value].push(newMessage)
  
  draft.value = ''
  const chat = chats.value.find(c => c.id === selectedId.value); 
  if (chat) { 
    chat.last = newMessage.text; 
    chat.time = newMessage.time 
  }
}


export function useChatData(){
  return { 
    isLinked, 
    qrSrc, 
    linkWithQR, 
    linkWithPhone, 
    disconnect, 
    checkExistingConnection,
    connectionStatus, 
    pairingCode, 
    phoneNumber,
    chats, 
    messagesByChat, 
    selectedId, 
    q, 
    draft, 
    showInfo,
    activeChat, 
    activeMessages, 
    totalUnread, 
    filteredChats,
    selectChat, 
    sendMessage,
    loadChats,
    loadMessages
  }
}