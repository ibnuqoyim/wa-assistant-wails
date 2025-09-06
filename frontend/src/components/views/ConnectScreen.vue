<template>
    <div class="connect-wrap">
    <div class="connect-center">
    <div class="connect-card">
    <div>
    <h3 class="connect-title">To set up WhatsApp on your computer</h3>
    
    <!-- Loading state -->
    <div v-if="connectionStatus === 'checking'" class="connect-loading">
      <div class="spinner"></div>
      <p>Checking for existing connection...</p>
    </div>
    
    <!-- Connecting state -->
    <div v-else-if="connectionStatus === 'connecting'" class="connect-loading">
      <div class="spinner"></div>
      <p>Connecting to WhatsApp...</p>
    </div>
    
    <!-- Connection options -->
    <div v-else class="connect-steps">
    <ol>
    <li>Open WhatsApp on your phone</li>
    <li>Tap <b>Menu</b> on Android, or <b>Settings</b> on iPhone</li>
    <li>Tap <b>Linked devices</b> and then <b>Link a device</b></li>
    <li>Point your phone at this screen to capture the QR code</li>
    </ol>
    <div class="connect-actions">
      <button class="connect-btn" @click="$emit('link-qr')" :disabled="connectionStatus === 'connecting'">
        {{ connectionStatus === 'connecting' ? 'Generating QR...' : 'Generate QR Code' }}
      </button>
      <div class="connect-link" @click="$emit('link-phone')">Link with phone number</div>
    </div>
    <div class="connect-footnote">You may need the latest version of WhatsApp.</div>
    
    <!-- Show pairing code if available -->
    <div v-if="pairingCode" class="pairing-code">
      <h4>Your pairing code:</h4>
      <div class="code">{{ pairingCode }}</div>
      <p>Enter this code in WhatsApp on your phone</p>
    </div>
    </div>
    </div>
    <div class="qr-box">
    <div class="qr">
    <img v-if="qrSrc" :src="qrSrc" alt="QR Code" class="qr-image" />
    <div v-else-if="connectionStatus === 'connecting'" class="qr-loading">
      <div class="spinner"></div>
      <p>Generating QR code...</p>
    </div>
    <div v-else class="qr-placeholder">
      <div class="qr-icon">📱</div>
      <p>QR code will appear here</p>
      <p style="font-size: 12px; color: #999;">Status: {{ connectionStatus }}, QR: {{ qrSrc ? 'Available' : 'None' }}</p>
    </div>
    </div>
    </div>
    </div>
    </div>
    </div>
    </template>
    <script setup>
    import { watch } from 'vue'
    
    // Props
    const props = defineProps({
      connectionStatus: {
        type: String,
        default: 'disconnected'
      },
      qrSrc: {
        type: String,
        default: ''
      },
      pairingCode: {
        type: String,
        default: ''
      }
    })
    
    // Debug: Watch for changes in props
    watch(() => props.qrSrc, (newVal) => {
      console.log('ConnectScreen: qrSrc changed to:', newVal ? newVal.substring(0, 50) + '...' : 'empty')
    })
    
    watch(() => props.connectionStatus, (newVal) => {
      console.log('ConnectScreen: connectionStatus changed to:', newVal)
    })
    
    // Emits: link-qr, link-phone
    defineEmits(['link-qr', 'link-phone'])
    </script>
    
    <style scoped>
    .connect-loading {
      text-align: center;
      padding: 2rem;
    }
    
    .spinner {
      width: 40px;
      height: 40px;
      border: 4px solid #f3f3f3;
      border-top: 4px solid #25d366;
      border-radius: 50%;
      animation: spin 1s linear infinite;
      margin: 0 auto 1rem;
    }
    
    @keyframes spin {
      0% { transform: rotate(0deg); }
      100% { transform: rotate(360deg); }
    }
    
    .connect-actions {
      margin: 1rem 0;
      text-align: center;
    }
    
    .connect-btn {
      background: #25d366;
      color: white;
      border: none;
      padding: 12px 24px;
      border-radius: 8px;
      font-size: 1rem;
      cursor: pointer;
      margin-bottom: 1rem;
      transition: background 0.3s;
    }
    
    .connect-btn:hover:not(:disabled) {
      background: #128c7e;
    }
    
    .connect-btn:disabled {
      background: #ccc;
      cursor: not-allowed;
    }
    
    .pairing-code {
      margin-top: 1rem;
      padding: 1rem;
      background: #f0f0f0;
      border-radius: 8px;
      text-align: center;
    }
    
    .code {
      font-size: 2rem;
      font-weight: bold;
      color: #25d366;
      margin: 0.5rem 0;
      letter-spacing: 0.2em;
    }
    
    .qr-placeholder, .qr-loading {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      height: 100%;
      color: #666;
    }
    
    .qr-icon {
      font-size: 3rem;
      margin-bottom: 1rem;
    }
    
    .qr-image {
      width: 100%;
      height: 100%;
      object-fit: contain;
      background: white;
      border-radius: 8px;
      padding: 10px;
      box-shadow: 0 2px 8px rgba(0,0,0,0.1);
    }
    
    .qr {
      position: relative;
      width: 256px;
      height: 256px;
      margin: 0 auto;
      border: 2px solid #e0e0e0;
      border-radius: 12px;
      background: #f9f9f9;
    }
    </style>