<template>
  <div class="settings-screen">
    <header class="settings-header">
      <h2>Auto-Reply Settings</h2>
      <p>Configure AI-powered automatic replies for WhatsApp messages</p>
    </header>

    <div class="settings-content">
      <!-- Enable/Disable Auto-Reply -->
      <div class="setting-group">
        <div class="setting-item">
          <label class="switch">
            <input 
              type="checkbox" 
              v-model="config.enabled"
              @change="saveConfig"
            >
            <span class="slider"></span>
          </label>
          <div class="setting-info">
            <h3>Enable Auto-Reply</h3>
            <p>Automatically respond to WhatsApp messages using AI</p>
          </div>
        </div>
      </div>

      <!-- AI Provider Selection -->
      <div class="setting-group">
        <h3>AI Provider</h3>
        <div class="radio-group">
          <label class="radio-item">
            <input 
              type="radio" 
              value="openai" 
              v-model="config.ai_provider"
              @change="saveConfig"
            >
            <span>OpenAI (GPT)</span>
          </label>
          <label class="radio-item">
            <input 
              type="radio" 
              value="ollama" 
              v-model="config.ai_provider"
              @change="saveConfig"
            >
            <span>Ollama (Local AI)</span>
          </label>
        </div>
      </div>

      <!-- OpenAI Settings -->
      <div v-if="config.ai_provider === 'openai'" class="setting-group">
        <h3>OpenAI Configuration</h3>
        <div class="input-group">
          <label>API Key</label>
          <input 
            type="password" 
            v-model="config.openai_api_key"
            placeholder="sk-..."
            @blur="saveConfig"
          >
        </div>
        <div class="input-group">
          <label>Model</label>
          <select v-model="config.openai_model" @change="saveConfig">
            <option value="gpt-3.5-turbo">GPT-3.5 Turbo</option>
            <option value="gpt-4">GPT-4</option>
            <option value="gpt-4-turbo">GPT-4 Turbo</option>
          </select>
        </div>
        <button @click="testConnection('openai')" class="test-btn" :disabled="testing">
          {{ testing ? 'Testing...' : 'Test OpenAI Connection' }}
        </button>
      </div>

      <!-- Ollama Settings -->
      <div v-if="config.ai_provider === 'ollama'" class="setting-group">
        <h3>Ollama Configuration</h3>
        <div class="input-group">
          <label>Ollama URL</label>
          <input 
            type="url" 
            v-model="config.ollama_url"
            placeholder="http://localhost:11434"
            @blur="saveConfig"
          >
        </div>
        <div class="input-group">
          <label>Model</label>
          <input 
            type="text" 
            v-model="config.ollama_model"
            placeholder="llama2"
            @blur="saveConfig"
          >
        </div>
        <button @click="testConnection('ollama')" class="test-btn" :disabled="testing">
          {{ testing ? 'Testing...' : 'Test Ollama Connection' }}
        </button>
      </div>

      <!-- Whitelist Settings -->
      <div class="setting-group">
        <h3>Whitelist Numbers</h3>
        <p>Only reply to messages from these phone numbers (without country code)</p>
        <div class="whitelist-container">
          <div v-for="(number, index) in config.whitelist_numbers" :key="index" class="whitelist-item">
            <input 
              type="tel" 
              v-model="config.whitelist_numbers[index]"
              placeholder="81234567890"
              @input="updateWhitelist"
              pattern="[0-9]*"
              inputmode="numeric"
            >
            <button @click="removeNumber(index)" class="remove-btn" type="button">×</button>
          </div>
          <button @click="addNumber" class="add-btn" type="button">+ Add Number</button>
        </div>
      </div>

      <!-- System Prompt -->
      <div class="setting-group">
        <h3>System Prompt</h3>
        <p>Instructions for the AI on how to respond</p>
        <textarea 
          v-model="config.system_prompt"
          rows="4"
          placeholder="You are a helpful WhatsApp assistant..."
          @blur="saveConfig"
        ></textarea>
      </div>

      <!-- Response Settings -->
      <div class="setting-group">
        <h3>Response Settings</h3>
        <div class="input-group">
          <label>Response Delay (seconds)</label>
          <input 
            type="number" 
            v-model.number="config.response_delay"
            min="0"
            max="60"
            @blur="saveConfig"
          >
        </div>
        <div class="input-group">
          <label>Max Response Length</label>
          <input 
            type="number" 
            v-model.number="config.max_response_length"
            min="50"
            max="2000"
            @blur="saveConfig"
          >
        </div>
      </div>

      <!-- Test Result -->
      <div v-if="testResult" class="test-result" :class="testResult.success ? 'success' : 'error'">
        <h4>{{ testResult.success ? 'Success!' : 'Error' }}</h4>
        <p>{{ testResult.message }}</p>
      </div>

      <!-- Save Configuration Button -->
      <div class="setting-group">
        <button 
          @click="saveConfiguration" 
          class="save-config-btn"
          :disabled="isSaving"
        >
          {{ isSaving ? 'Saving...' : 'Save Configuration' }}
        </button>
        <div v-if="saveResult" class="save-result" :class="saveResult.success ? 'success' : 'error'">
          {{ saveResult.message }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { GetAutoReplyConfig, UpdateAutoReplyConfig, TestAIConnection } from '../../../wailsjs/go/main/App'

interface AutoReplyConfig {
  enabled: boolean
  ai_provider: string
  openai_api_key: string
  openai_model: string
  ollama_url: string
  ollama_model: string
  whitelist_numbers: string[]
  system_prompt: string
  response_delay: number
  max_response_length: number
}

const config = ref<AutoReplyConfig>({
  enabled: false,
  ai_provider: 'openai',
  openai_api_key: '',
  openai_model: 'gpt-3.5-turbo',
  ollama_url: 'http://localhost:11434',
  ollama_model: 'llama2',
  whitelist_numbers: [], // This will be populated from backend
  system_prompt: 'You are a helpful WhatsApp assistant. Respond briefly and helpfully to messages.',
  response_delay: 2,
  max_response_length: 500
})

// Validate phone number format
const validatePhoneNumber = (number: string): boolean => {
  // Remove any non-digit characters
  const digits = number.replace(/\D/g, '')
  // Check if it's a valid length (adjust min/max as needed)
  return digits.length >= 10 && digits.length <= 15
}

const testing = ref(false)
const testResult = ref<{success: boolean, message: string} | null>(null)

const isSaving = ref(false)
const saveResult = ref<{success: boolean, message: string} | null>(null)

// Function to save configuration to persistent storage
const saveConfiguration = async () => {
  isSaving.value = true
  saveResult.value = null
  
  try {
    await UpdateAutoReplyConfig(config.value)
    saveResult.value = {
      success: true,
      message: 'Configuration saved successfully'
    }
  } catch (error) {
    saveResult.value = {
      success: false,
      message: `Failed to save configuration: ${error}`
    }
  } finally {
    isSaving.value = false
    // Hide success message after 3 seconds
    if (saveResult.value?.success) {
      setTimeout(() => {
        saveResult.value = null
      }, 3000)
    }
  }
}

onMounted(async () => {
  try {
    const result = await GetAutoReplyConfig()
    if (result) {
      config.value = result
    }
  } catch (error) {
    console.error('Failed to load auto-reply config:', error)
  }
})

const saveConfig = async () => {
  try {
    await UpdateAutoReplyConfig(config.value)
    console.log('Config saved successfully')
  } catch (error) {
    console.error('Failed to save config:', error)
  }
}

const testConnection = async (provider: string) => {
  testing.value = true
  testResult.value = null
  
  try {
    const response = await TestAIConnection(provider)
    testResult.value = {
      success: true,
      message: `Connection successful! Response: "${response}"`
    }
  } catch (error) {
    testResult.value = {
      success: false,
      message: `Connection failed: ${error}`
    }
  } finally {
    testing.value = false
  }
}

const addNumber = async () => {
  if (!config.value.whitelist_numbers) {
    config.value.whitelist_numbers = []
  }
  config.value.whitelist_numbers.push('')
  // Save after adding to ensure it's persisted to backend
  await saveConfig()
}

const removeNumber = async (index: number) => {
  config.value.whitelist_numbers.splice(index, 1)
  await saveConfig()
}

// Watch for changes in individual whitelist numbers
const updateWhitelist = async () => {
  try {
    // Filter out any empty numbers before saving
    config.value.whitelist_numbers = config.value.whitelist_numbers.filter(number => number.trim() !== '')
    
    // Validate all numbers
    const invalidNumbers = config.value.whitelist_numbers.filter(number => !validatePhoneNumber(number))
    if (invalidNumbers.length > 0) {
      console.warn('Invalid phone numbers detected:', invalidNumbers)
      // You might want to show an error message to the user here
      return
    }
    
    await saveConfig()
  } catch (error) {
    console.error('Failed to save whitelist:', error)
  }
}
</script>

<style scoped>
.settings-screen {
  height: 100vh;
  max-height: 100vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  background: var(--panel);
}

.settings-header {
  padding: 24px 32px;
  background: var(--panel-2);
  border-bottom: 1px solid var(--panel-3);
  position: sticky;
  top: 0;
  z-index: 10;
}

.settings-header h2 {
  color: var(--text);
  margin-bottom: 8px;
  font-size: 24px;
}

.settings-header p {
  color: var(--muted);
  font-size: 14px;
}

.settings-content {
  flex: 1;
  overflow-y: auto;
  padding: 24px 32px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.settings-content::-webkit-scrollbar {
  width: 6px;
}

.settings-content::-webkit-scrollbar-track {
  background: transparent;
}

.settings-content::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 3px;
}

.settings-content::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.2);
}

.setting-group {
  background: var(--panel-2);
  padding: 24px;
  border-radius: 16px;
  border: 1px solid var(--panel-3);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  transition: transform 0.2s, box-shadow 0.2s;
}

.setting-group:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
}

.setting-group h3 {
  color: var(--text);
  margin-bottom: 16px;
  font-size: 18px;
  font-weight: 600;
}

.setting-group p {
  color: var(--muted);
  font-size: 14px;
  margin-bottom: 20px;
  line-height: 1.5;
}

.setting-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
  background: var(--panel);
  border-radius: 12px;
  margin-bottom: 16px;
}

/* Switch Toggle */
.switch {
  position: relative;
  display: inline-block;
  width: 52px;
  height: 26px;
  flex-shrink: 0;
}

.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--panel-3);
  transition: .3s;
  border-radius: 26px;
}

.slider:before {
  position: absolute;
  content: "";
  height: 20px;
  width: 20px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  transition: .3s;
  border-radius: 50%;
}

input:checked + .slider {
  background-color: var(--brand);
}

input:checked + .slider:before {
  transform: translateX(26px);
}

/* Radio Group */
.radio-group {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.radio-item {
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  padding: 12px 16px;
  border-radius: 12px;
  background: var(--panel);
  transition: all 0.2s;
}

.radio-item:hover {
  background: var(--hover);
  transform: translateY(-1px);
}

.radio-item input[type="radio"] {
  width: 20px;
  height: 20px;
  margin: 0;
  cursor: pointer;
}

/* Input Groups */
.input-group {
  margin-bottom: 20px;
}

.input-group label {
  display: block;
  color: var(--text);
  font-weight: 500;
  margin-bottom: 8px;
  font-size: 14px;
}

.input-group input,
.input-group select,
.input-group textarea {
  width: 100%;
  padding: 12px 16px;
  border: 2px solid var(--panel-3);
  border-radius: 12px;
  background: var(--panel);
  color: var(--text);
  font-size: 14px;
  transition: all 0.2s;
}

.input-group input:hover,
.input-group select:hover,
.input-group textarea:hover {
  border-color: var(--brand);
}

.input-group input:focus,
.input-group select:focus,
.input-group textarea:focus {
  outline: none;
  border-color: var(--brand);
  box-shadow: 0 0 0 3px rgba(5, 150, 105, 0.1);
}

/* Whitelist */
.whitelist-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.whitelist-item {
  display: flex;
  gap: 12px;
  align-items: center;
}

.whitelist-item input {
  flex: 1;
}

.remove-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: var(--error);
  color: white;
  border-radius: 50%;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  transition: all 0.2s;
}

.remove-btn:hover {
  background: #dc2626;
  transform: scale(1.1);
}

.add-btn {
  padding: 12px 16px;
  border: 2px dashed var(--panel-3);
  background: transparent;
  color: var(--muted);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
  font-weight: 500;
}

.add-btn:hover {
  border-color: var(--brand);
  color: var(--brand);
  transform: translateY(-1px);
}

/* Test Button */
.test-btn {
  width: 100%;
  padding: 12px 24px;
  background: var(--brand);
  color: white;
  border: none;
  border-radius: 12px;
  cursor: pointer;
  font-weight: 600;
  font-size: 14px;
  transition: all 0.2s;
}

.test-btn:hover:not(:disabled) {
  background: #059669;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(5, 150, 105, 0.2);
}

.test-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Test Result */
.test-result {
  padding: 16px;
  border-radius: 12px;
  margin-top: 16px;
  transition: all 0.2s;
}

.test-result.success {
  background: #d1fae5;
  border: 2px solid #10b981;
  color: #065f46;
}

.test-result.error {
  background: #fee2e2;
  border: 2px solid #ef4444;
  color: #991b1b;
}

.test-result h4 {
  margin: 0 0 8px 0;
  font-size: 16px;
  font-weight: 600;
}

.test-result p {
  margin: 0;
  font-size: 14px;
  line-height: 1.5;
}

/* Save Configuration Button */
.save-config-btn {
  width: 100%;
  padding: 12px 20px;
  background: var(--brand);
  color: white;
  border: none;
  border-radius: 6px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  margin-top: 10px;
}

.save-config-btn:hover:not(:disabled) {
  background: #059669;
}

.save-config-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.save-result {
  margin-top: 10px;
  padding: 10px;
  border-radius: 6px;
  text-align: center;
  font-size: 14px;
  transition: opacity 0.3s;
}

.save-result.success {
  background: #d1fae5;
  color: #065f46;
  border: 1px solid #10b981;
}

.save-result.error {
  background: #fee2e2;
  color: #991b1b;
  border: 1px solid #ef4444;
}
</style>
