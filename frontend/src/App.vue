<template>
    <div v-if="!isLinked" class="connect-wrap">
    <main class="main-content">
      <ConnectScreen 
        v-if="connectionStatus === 'checking' || connectionStatus === 'connecting'"
        :connection-status="connectionStatus"
        :qr-src="qrSrc"
        @link-qr="handleLinkQR"
        @link-phone="handleLinkPhone"
      />
      <div v-if="connectionStatus === 'connected'" class="main-content">
      <!-- Chat View -->
      <div v-if="currentView === 'chat'" class="chat-container">
        <ChatList
          :chats="filteredChats"
          :selected-id="selectedId"
          @select="selectChat"
        />
        <ChatWindow
          v-if="activeChat"
          :chat="activeChat"
          :messages="activeMessages"
          :draft="draft"
          :show-info="showInfo"
          @send="sendMessage"
          @update:draft="draft = $event"
          @update:show-info="showInfo = $event"
        />
      </div>
      
      <!-- Contacts View -->
      <ContactsScreen 
        v-else-if="currentView === 'contacts'" 
        @start-chat="handleStartChat"
      />
      
      <!-- Settings View -->
      <SettingsScreen v-else-if="currentView === 'settings'" />
    </div>
      <div v-else class="error-state">
        <p>Connection failed. Please try again.</p>
        <button @click="checkConnection">Retry</button>
      </div>
    </main>

    <!-- Navigation Bar -->
    <nav v-if="connectionStatus === 'connected'" class="bottom-nav">
      <button 
        @click="currentView = 'chat'" 
        :class="{ active: currentView === 'chat' }"
        class="nav-btn"
      >
        <svg viewBox="0 0 24 24" class="nav-icon">
          <path d="M20 2H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h4l4 4 4-4h4c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2z" fill="currentColor"/>
        </svg>
        <span>Chats</span>
      </button>
      <button 
        @click="currentView = 'contacts'" 
        :class="{ active: currentView === 'contacts' }"
        class="nav-btn"
      >
        <svg viewBox="0 0 24 24" class="nav-icon">
          <path d="M16 4c0-1.11.89-2 2-2s2 .89 2 2-.89 2-2 2-2-.89-2-2M4 18v-1c0-1.1.9-2 2-2h2c1.1 0 2 .9 2 2v1h2v-1c0-1.1.9-2 2-2h2c1.1 0 2 .9 2 2v1h2c1.1 0 2-.9 2-2v-3H2v3c0 1.1.9 2 2 2h2M18 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2m-8 0c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2M6 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2" fill="currentColor"/>
        </svg>
        <span>Contacts</span>
      </button>
      <button 
        @click="currentView = 'settings'" 
        :class="{ active: currentView === 'settings' }"
        class="nav-btn"
      >
        <svg viewBox="0 0 24 24" class="nav-icon">
          <path d="M12 15.5A3.5 3.5 0 0 1 8.5 12A3.5 3.5 0 0 1 12 8.5a3.5 3.5 0 0 1 3.5 3.5 3.5 3.5 0 0 1-3.5 3.5m7.43-2.53c.04-.32.07-.64.07-.97 0-.33-.03-.66-.07-1l2.11-1.63c.19-.15.24-.42.12-.64l-2-3.46c-.12-.22-.39-.31-.61-.22l-2.49 1c-.52-.39-1.06-.73-1.69-.98l-.37-2.65A.506.506 0 0 0 14 2h-4c-.25 0-.46.18-.5.42l-.37 2.65c-.63.25-1.17.59-1.69.98l-2.49-1c-.22-.09-.49 0-.61.22l-2 3.46c-.13.22-.07.49.12.64L4.57 11c-.04.34-.07.67-.07 1 0 .33.03.65.07.97l-2.11 1.66c-.19.15-.25.42-.12.64l2 3.46c.12.22.39.3.61.22l2.49-1.01c.52.4 1.06.74 1.69.99l.37 2.65c.04.24.25.42.5.42h4c.25 0 .46-.18.5-.42l.37-2.65c.63-.26 1.17-.59 1.69-.99l2.49 1.01c.22.08.49 0 .61-.22l2-3.46c.12-.22.07-.49-.12-.64l-2.11-1.66Z" fill="currentColor"/>
        </svg>
        <span>Settings</span>
      </button>
    </nav>
    </div>
    
    
    <div v-else class="wa-app">
    <Sidebar 
      :total-unread="totalUnread" 
      :current-view="currentView"
      @view-change="currentView = $event"
    />
    
    <!-- Chat View -->
    <div v-if="currentView === 'chat'" class="chat-container">
      <ChatList
        :chats="filteredChats"
        :selected-id="selectedId"
        @select="selectChat"
      />
      <ChatWindow
        v-if="activeChat"
        :chat="activeChat"
        :messages="activeMessages"
        :draft="draft"
        :show-info="showInfo"
        @send="sendMessage"
        @update:draft="draft = $event"
        @update:show-info="showInfo = $event"
      />
    </div>
    
    <!-- Contacts View -->
    <ContactsScreen 
      v-else-if="currentView === 'contacts'" 
      @start-chat="handleStartChat"
    />
    
    <!-- Settings View -->
    <SettingsScreen v-else-if="currentView === 'settings'" />
    </div>
    </template>
    
    
    <script setup>
    import { onMounted, ref } from 'vue'
    import { useChatData } from './composables/useChatData'
    import Sidebar from './components/views/Sidebar.vue'
    import ChatList from './components/views/ChatList.vue'
    import ChatWindow from './components/views/ChatWindow.vue'
    import ConnectScreen from './components/views/ConnectScreen.vue'
    import SettingsScreen from './components/views/SettingsScreen.vue'
    import ContactsScreen from './components/views/ContactsScreen.vue'

    const currentView = ref('chat')

    const { isLinked, linkWithQR, linkWithPhone, filteredChats, q, selectedId, selectChat, activeChat, activeMessages, draft, showInfo, sendMessage, totalUnread, connectionStatus, qrSrc, pairingCode, checkExistingConnection } = useChatData()

    const handleLinkQR = () => {
      linkWithQR()
    }

    const handleLinkPhone = (phoneNumber) => {
      linkWithPhone(phoneNumber)
    }

    const handleStartChat = (jid) => {
      const chat = filteredChats.value.find(c => c.id === jid)
      if (chat) {
        selectChat(chat.id)
        currentView.value = 'chat'
      }
    }

    const checkConnection = () => {
      checkExistingConnection()
    }
    
    // Check for existing connection when app starts
    onMounted(() => {
      checkExistingConnection()
    })
    </script>


<style scoped>
.wa-app{display:grid;grid-template-columns:72px 360px 1fr;height:100vh;background:var(--bg);color:var(--text)}

.connect-wrap {
  height: 100vh;
  display: flex;
  flex-direction: column;
}

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.chat-container {
  display: grid;
  grid-template-columns: 360px 1fr;
  height: 100vh;
}

.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100vh;
  gap: 20px;
}

.error-state button {
  padding: 10px 20px;
  background: var(--brand);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
}

/* Bottom Navigation */
.bottom-nav {
  display: flex;
  background: var(--panel);
  border-top: 1px solid var(--panel-3);
  padding: 8px;
  gap: 4px;
}

.nav-btn {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 8px 12px;
  background: transparent;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  color: var(--muted);
  transition: all 0.2s;
}

.nav-btn:hover {
  background: var(--hover);
  color: var(--text);
}

.nav-btn.active {
  background: var(--brand);
  color: white;
}

.nav-icon {
  width: 20px;
  height: 20px;
}

.nav-btn span {
  font-size: 12px;
  font-weight: 500;
}
</style>