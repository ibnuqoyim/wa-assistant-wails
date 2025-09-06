<template>
  <div class="contacts-screen">
    <!-- Header -->
    <div class="contacts-header">
      <h2>Contacts</h2>
      <div class="search-box">
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search contacts..."
          class="search-input"
        />
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>Loading contacts...</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="error-state">
      <p>{{ error }}</p>
      <button @click="loadContacts" class="retry-btn">Retry</button>
    </div>

    <!-- Contacts List -->
    <div v-else class="contacts-list">
      <div class="contacts-stats">
        <span>{{ filteredContacts.length }} contacts</span>
        <div class="filter-buttons">
          <button 
            @click="filterType = 'all'" 
            :class="{ active: filterType === 'all' }"
            class="filter-btn"
          >
            All
          </button>
          <button 
            @click="filterType = 'personal'" 
            :class="{ active: filterType === 'personal' }"
            class="filter-btn"
          >
            Personal
          </button>
          <button 
            @click="filterType = 'groups'" 
            :class="{ active: filterType === 'groups' }"
            class="filter-btn"
          >
            Groups
          </button>
          <button 
            @click="filterType = 'business'" 
            :class="{ active: filterType === 'business' }"
            class="filter-btn"
          >
            Business
          </button>
        </div>
      </div>

      <div class="contacts-container">
        <div
          v-for="contact in filteredContacts"
          :key="contact.jid"
          @click="selectContact(contact)"
          :class="{ active: selectedContact?.jid === contact.jid }"
          class="contact-item"
        >
          <div class="contact-avatar">
            <img
              v-if="contact.profilePicUrl"
              :src="contact.profilePicUrl"
              :alt="contact.name"
              class="avatar-img"
            />
            <div v-else class="avatar-placeholder">
              {{ getInitials(contact.name) }}
            </div>
          </div>
          
          <div class="contact-info">
            <div class="contact-name">{{ contact.name }}</div>
            <div class="contact-details">
              <span v-if="contact.phoneNumber" class="phone">{{ contact.phoneNumber }}</span>
              <span v-if="contact.isGroup" class="group-badge">Group</span>
              <span v-if="contact.isBusiness" class="business-badge">Business</span>
            </div>
          </div>

          <div class="contact-actions">
            <button @click.stop="startChat(contact)" class="action-btn chat-btn">
              <svg class="icon" viewBox="0 0 24 24">
                <path d="M20 2H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h4l4 4 4-4h4c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2z"/>
              </svg>
            </button>
            <button @click.stop="viewContactInfo(contact)" class="action-btn info-btn">
              <svg class="icon" viewBox="0 0 24 24">
                <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/>
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Contact Info Modal -->
    <div v-if="showContactInfo && contactInfo" class="modal-overlay" @click="closeContactInfo">
      <div class="contact-info-modal" @click.stop>
        <div class="modal-header">
          <h3>Contact Information</h3>
          <button @click="closeContactInfo" class="close-btn">×</button>
        </div>
        
        <div class="modal-content">
          <div class="contact-profile">
            <div class="profile-avatar">
              <img
                v-if="contactInfo.profilePicUrl"
                :src="contactInfo.profilePicUrl"
                :alt="contactInfo.name"
                class="profile-img"
              />
              <div v-else class="profile-placeholder">
                {{ getInitials(contactInfo.name) }}
              </div>
            </div>
            <div class="profile-info">
              <h4>{{ contactInfo.name }}</h4>
              <p v-if="contactInfo.phoneNumber">{{ contactInfo.phoneNumber }}</p>
              <p v-if="contactInfo.status" class="status">{{ contactInfo.status }}</p>
            </div>
          </div>

          <div class="contact-details-list">
            <div v-if="contactInfo.pushName" class="detail-item">
              <span class="label">Push Name:</span>
              <span class="value">{{ contactInfo.pushName }}</span>
            </div>
            <div v-if="contactInfo.businessName" class="detail-item">
              <span class="label">Business Name:</span>
              <span class="value">{{ contactInfo.businessName }}</span>
            </div>
            <div class="detail-item">
              <span class="label">Type:</span>
              <span class="value">
                {{ contactInfo.isGroup ? 'Group' : contactInfo.isBusiness ? 'Business' : 'Personal' }}
              </span>
            </div>
            <div v-if="contactInfo.lastSeen" class="detail-item">
              <span class="label">Last Seen:</span>
              <span class="value">{{ contactInfo.lastSeen }}</span>
            </div>
          </div>

          <!-- Group Info -->
          <div v-if="contactInfo.groupInfo" class="group-info">
            <h5>Group Information</h5>
            <div class="detail-item">
              <span class="label">Description:</span>
              <span class="value">{{ contactInfo.groupInfo.description || 'No description' }}</span>
            </div>
            <div class="detail-item">
              <span class="label">Members:</span>
              <span class="value">{{ contactInfo.groupInfo.memberCount }}</span>
            </div>
            <div class="detail-item">
              <span class="label">Created:</span>
              <span class="value">{{ formatDate(contactInfo.groupInfo.createdAt) }}</span>
            </div>
          </div>
        </div>

        <div class="modal-actions">
          <button @click="startChat(contactInfo)" class="primary-btn">Start Chat</button>
          <button @click="closeContactInfo" class="secondary-btn">Close</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { GetContacts, GetContactInfo } from '../../../wailsjs/go/main/App'

const contacts = ref([])
const loading = ref(false)
const error = ref('')
const searchQuery = ref('')
const filterType = ref('all')
const selectedContact = ref(null)
const showContactInfo = ref(false)
const contactInfo = ref(null)

const emit = defineEmits(['start-chat'])

// Computed properties
const filteredContacts = computed(() => {
  let filtered = contacts.value

  // Filter by type
  if (filterType.value === 'personal') {
    filtered = filtered.filter(c => !c.isGroup && !c.isBusiness)
  } else if (filterType.value === 'groups') {
    filtered = filtered.filter(c => c.isGroup)
  } else if (filterType.value === 'business') {
    filtered = filtered.filter(c => c.isBusiness)
  }

  // Filter by search query
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    filtered = filtered.filter(c => 
      c.name.toLowerCase().includes(query) ||
      (c.phoneNumber && c.phoneNumber.includes(query)) ||
      (c.pushName && c.pushName.toLowerCase().includes(query))
    )
  }

  return filtered.sort((a, b) => a.name.localeCompare(b.name))
})

// Methods
const loadContacts = async () => {
  loading.value = true
  error.value = ''
  
  try {
    const result = await GetContacts()
    contacts.value = result || []
  } catch (err) {
    error.value = `Failed to load contacts: ${err.message || err}`
    console.error('Error loading contacts:', err)
  } finally {
    loading.value = false
  }
}

const selectContact = (contact) => {
  selectedContact.value = contact
}

const startChat = (contact) => {
  emit('start-chat', contact.jid)
}

const viewContactInfo = async (contact) => {
  try {
    const info = await GetContactInfo(contact.jid)
    contactInfo.value = info
    showContactInfo.value = true
  } catch (err) {
    console.error('Error getting contact info:', err)
    // Fallback to basic contact info
    contactInfo.value = contact
    showContactInfo.value = true
  }
}

const closeContactInfo = () => {
  showContactInfo.value = false
  contactInfo.value = null
}

const getInitials = (name) => {
  if (!name) return '?'
  return name
    .split(' ')
    .map(word => word[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)
}

const formatDate = (dateStr) => {
  if (!dateStr) return 'Unknown'
  try {
    return new Date(dateStr).toLocaleDateString()
  } catch {
    return 'Unknown'
  }
}

// Load contacts on mount
onMounted(() => {
  loadContacts()
})
</script>

<style scoped>
.contacts-screen {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--bg);
}

.contacts-header {
  padding: 20px;
  border-bottom: 1px solid var(--panel-3);
  background: var(--panel);
}

.contacts-header h2 {
  margin: 0 0 16px 0;
  color: var(--text);
  font-size: 24px;
  font-weight: 600;
}

.search-box {
  position: relative;
}

.search-input {
  width: 100%;
  padding: 12px 16px;
  border: 1px solid var(--panel-3);
  border-radius: 8px;
  background: var(--bg);
  color: var(--text);
  font-size: 14px;
}

.search-input:focus {
  outline: none;
  border-color: var(--brand);
}

.loading-state, .error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  gap: 16px;
  color: var(--muted);
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--panel-3);
  border-top: 3px solid var(--brand);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.retry-btn {
  padding: 8px 16px;
  background: var(--brand);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
}

.contacts-list {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.contacts-stats {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--panel-3);
  background: var(--panel);
}

.contacts-stats span {
  color: var(--muted);
  font-size: 14px;
}

.filter-buttons {
  display: flex;
  gap: 8px;
}

.filter-btn {
  padding: 6px 12px;
  background: transparent;
  border: 1px solid var(--panel-3);
  border-radius: 6px;
  color: var(--muted);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.filter-btn:hover {
  background: var(--hover);
}

.filter-btn.active {
  background: var(--brand);
  color: white;
  border-color: var(--brand);
}

.contacts-container {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.contact-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  margin-bottom: 4px;
}

.contact-item:hover {
  background: var(--hover);
}

.contact-item.active {
  background: var(--brand-light);
}

.contact-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
}

.avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-placeholder {
  width: 100%;
  height: 100%;
  background: var(--brand);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 16px;
}

.contact-info {
  flex: 1;
  min-width: 0;
}

.contact-name {
  font-weight: 500;
  color: var(--text);
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.contact-details {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--muted);
}

.phone {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.group-badge, .business-badge {
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 500;
}

.group-badge {
  background: var(--success-light);
  color: var(--success);
}

.business-badge {
  background: var(--warning-light);
  color: var(--warning);
}

.contact-actions {
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.2s;
}

.contact-item:hover .contact-actions {
  opacity: 1;
}

.action-btn {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.chat-btn {
  background: var(--brand);
  color: white;
}

.info-btn {
  background: var(--panel-3);
  color: var(--muted);
}

.action-btn:hover {
  transform: scale(1.1);
}

.icon {
  width: 16px;
  height: 16px;
  fill: currentColor;
}

/* Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.contact-info-modal {
  background: var(--panel);
  border-radius: 12px;
  width: 90%;
  max-width: 500px;
  max-height: 80vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid var(--panel-3);
}

.modal-header h3 {
  margin: 0;
  color: var(--text);
}

.close-btn {
  background: none;
  border: none;
  font-size: 24px;
  color: var(--muted);
  cursor: pointer;
  padding: 0;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-content {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.contact-profile {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
}

.profile-avatar {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  overflow: hidden;
}

.profile-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.profile-placeholder {
  width: 100%;
  height: 100%;
  background: var(--brand);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 24px;
}

.profile-info h4 {
  margin: 0 0 8px 0;
  color: var(--text);
  font-size: 20px;
}

.profile-info p {
  margin: 0;
  color: var(--muted);
}

.status {
  font-style: italic;
}

.contact-details-list {
  margin-bottom: 24px;
}

.detail-item {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid var(--panel-3);
}

.detail-item:last-child {
  border-bottom: none;
}

.label {
  font-weight: 500;
  color: var(--muted);
}

.value {
  color: var(--text);
  text-align: right;
}

.group-info {
  border-top: 1px solid var(--panel-3);
  padding-top: 16px;
}

.group-info h5 {
  margin: 0 0 16px 0;
  color: var(--text);
}

.modal-actions {
  display: flex;
  gap: 12px;
  padding: 20px;
  border-top: 1px solid var(--panel-3);
}

.primary-btn {
  flex: 1;
  padding: 12px;
  background: var(--brand);
  color: white;
  border: none;
  border-radius: 8px;
  font-weight: 500;
  cursor: pointer;
}

.secondary-btn {
  flex: 1;
  padding: 12px;
  background: transparent;
  color: var(--muted);
  border: 1px solid var(--panel-3);
  border-radius: 8px;
  font-weight: 500;
  cursor: pointer;
}

.primary-btn:hover {
  background: var(--brand-dark);
}

.secondary-btn:hover {
  background: var(--hover);
}
</style>
