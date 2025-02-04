<script lang="ts">
  import { onMount, afterUpdate, tick } from 'svelte';
  import ChatMessage from './ChatMessage.svelte';

  type Message = {
    ID: number;
    ChatID: number;
    Role: string;
    Content: string;
    CreatedAt: string;
    UpdatedAt: string;
    DeletedAt: string | null;
    ModelName: string;
    Starred: boolean;
  };

  // Add this new type for messages being composed
  type NewMessage = {
    Role: string;
    Content: string;
    ID?: number;  // Add optional ID field
    ModelName?: string;  // Add this field
  };

  type Chat = {
    ID: number;
    Messages: Message[];
    CreatedAt: string;
    UpdatedAt: string;
    DeletedAt: string | null;
    ModelName: string;  // Updated from Model to ModelName
    Starred: boolean;
  };

  type ModelPricing = {
    prompt: string;
    completion: string;
    image: string;
    request: string;
  };

  type ModelArchitecture = {
    modality: string;
    tokenizer: string;
    instruct_type: string | null;
  };

  type OpenRouterModel = {
    id: string;
    name: string;
    description?: string;
    pricing: ModelPricing;
    architecture: ModelArchitecture;
  };

  let availableModels: Record<string, OpenRouterModel> = {};
  let messages: (Message | NewMessage)[] = [];
  let userInput = '';
  let isLoading = false;
  let selectedModel = '';  // We'll set this after fetching models
  let searchTerm = '';
  let filteredModels: Record<string, OpenRouterModel> = {};
  let isDropdownOpen = false;
  let searchInput: HTMLInputElement;
  let previousChats: Chat[] = [];
  let showChatHistory = false;
  let currentChatId: number | null = null;
  let modelSelectorFocused = false;

  // Declare the reference to the messages container for sticky scrolling
  let messagesContainer: HTMLDivElement;

  let autoScroll = true; // Enable sticky scrolling by default

  // Auto-scroll to the bottom when messages update if autoScroll is enabled
  afterUpdate(() => {
    if (messagesContainer && autoScroll) {
      messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }
  });

  // Handle user scrolling to disable auto scrolling when the user scrolls up
  function handleScroll() {
    if (messagesContainer) {
      const { scrollTop, scrollHeight, clientHeight } = messagesContainer;
      // If the user scrolls up (beyond a small threshold), disable autoScrolling
      autoScroll = scrollTop + clientHeight >= scrollHeight - 50;
    }
  }

  // Function to manually scroll to the bottom and re-enable auto scrolling
  function scrollToBottom() {
    if (messagesContainer) {
      messagesContainer.scrollTop = messagesContainer.scrollHeight;
      autoScroll = true;
    }
  }

  onMount(async () => {
    const savedModel = localStorage.getItem('selectedModel');
    
    try {
      const response = await fetch('https://openrouter.ai/api/v1/models');
      
      if (!response.ok) throw new Error('Failed to fetch models');
      
      const data = await response.json();
      const openRouterModels: Record<string, OpenRouterModel> = {};
      
      data.data.forEach((model: OpenRouterModel) => {
        openRouterModels[model.id] = {
          id: model.id,
          name: model.name,
          description: model.description,
          pricing: model.pricing,
          architecture: model.architecture
        };
      });

      availableModels = openRouterModels;

      // Set default model to Claude 3 Haiku if available, otherwise use first available model
      if (savedModel && availableModels[savedModel]) {
        selectedModel = savedModel;
      } else {
        selectedModel = Object.keys(availableModels).find(id => 
          id === 'anthropic/claude-3.5-haiku'
        ) || Object.keys(availableModels)[0];
      }

      // Initialize filtered models
      filteredModels = { ...availableModels };
    } catch (error) {
      console.error('Error fetching OpenRouter models:', error);
      // Show error state or fallback
    }

    // Fetch chat history
    try {
      const response = await fetch('http://localhost:8088/api/history');
      if (response.ok) {
        const data = await response.json();
        // Convert the object of chats into an array
        previousChats = Object.values(data).map((chat: any) => ({
          ID: chat.id,
          Messages: chat.messages.map((msg: any) => ({
            ID: msg.id,
            ChatID: msg.chat_id,
            Role: msg.role,
            Content: msg.content,
            CreatedAt: msg.created_at,
            UpdatedAt: msg.updated_at,
            DeletedAt: msg.deleted_at,
            ModelName: msg.model_name,
            Starred: msg.starred
          })),
          CreatedAt: chat.created_at,
          UpdatedAt: chat.updated_at,
          DeletedAt: chat.deleted_at,
          ModelName: chat.model_name,
          Starred: chat.starred
        }));
      }
    } catch (error) {
      console.error('Error fetching chat history:', error);
    }
  });

  async function* streamResponse(reader: ReadableStreamDefaultReader<Uint8Array>) {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      const text = new TextDecoder().decode(value);
      const lines = text.split('\n').filter(line => line.trim() !== '');

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          try {
            const data = JSON.parse(line.slice(6));
            if (data.choices?.[0]?.delta?.content) {
              yield data.choices[0].delta.content;
            }
          } catch (e) {
            console.error('Error parsing JSON:', e);
          }
        }
      }
    }
  }

  async function updateChatHistory() {
    try {
        const historyResponse = await fetch('http://localhost:8088/api/history');
        if (historyResponse.ok) {
            const data = await historyResponse.json();
            previousChats = data.map((chat: any) => ({
                ID: chat.id,
                Messages: chat.messages.map((msg: any) => ({
                    ID: msg.id,
                    ChatID: msg.chat_id,
                    Role: msg.role,
                    Content: msg.content,
                    CreatedAt: msg.created_at,
                    UpdatedAt: msg.updated_at,
                    DeletedAt: msg.deleted_at,
                    ModelName: msg.model_name,
                    Starred: msg.starred
                })),
                CreatedAt: chat.created_at,
                UpdatedAt: chat.updated_at,
                DeletedAt: chat.deleted_at,
                ModelName: chat.model_name,
                Starred: chat.starred
            }));
        }
    } catch (error) {
        console.error('Error updating chat history:', error);
    }
  }

  async function handleSubmit() {
    if (!userInput.trim()) return;

    const newUserMessage = { 
        Role: 'user', 
        Content: userInput,
        ModelName: selectedModel
    };
    
    const maxRetries = 2;
    let retryCount = 0;
    let success = false;

    while (retryCount <= maxRetries && !success) {
        try {
            if (retryCount > 0) {
                console.log(`Retrying request (attempt ${retryCount + 1})`);
            }

            messages = [...messages, newUserMessage];
            const currentInput = userInput;
            userInput = '';
            isLoading = true;

            // Create new assistant message
            messages = [...messages, { 
                Role: 'assistant', 
                Content: '',
                ModelName: selectedModel 
            } as NewMessage];

            // Create the request body with the chat_id
            const requestBody: {
                model: string;
                messages: { id: number | undefined; role: string; content: string; }[];
                stream: boolean;
                chat_id?: number;  // Make chat_id optional
            } = {
                model: selectedModel,
                messages: messages.slice(0, -1).map(msg => ({
                    id: 'ID' in msg ? msg.ID : undefined,
                    role: msg.Role,
                    content: msg.Content
                })),
                stream: true
            };

            // Only include chat_id if it exists
            if (currentChatId !== null) {
                requestBody.chat_id = currentChatId;
            }

            const response = await fetch('http://localhost:8088/api/chat', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(requestBody),
            });

            if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);

            const reader = response.body?.getReader();
            if (!reader) throw new Error('No reader available');

            let hasContent = false;
            for await (const content of streamResponse(reader)) {
                if (content.trim()) {
                    hasContent = true;
                    messages[messages.length - 1].Content += content;
                    messages = messages;
                }
            }

            if (!hasContent) {
                // Remove empty message and retry
                messages = messages.slice(0, -2); // Remove both user and assistant messages
                if (retryCount === maxRetries) {
                    messages = [...messages, newUserMessage, {
                        Role: 'assistant',
                        Content: 'Sorry, the model returned empty responses. Please try again later.',
                        ModelName: selectedModel
                    }];
                }
                throw new Error('Empty response');
            }

            success = true;
            await updateChatHistory();

        } catch (error) {
            console.error(`Error (attempt ${retryCount + 1}):`, error);
            if (retryCount === maxRetries) {
                messages = messages.slice(0, -2); // Remove failed messages
                messages = [...messages, newUserMessage, {
                    Role: 'assistant',
                    Content: 'Sorry, there was an error processing your request after multiple attempts.',
                    ModelName: selectedModel
                }];
            }
            retryCount++;
        }
    }

    isLoading = false;
  }

  function handleModelSelect(modelId: string) {
    selectedModel = modelId;
    localStorage.setItem('selectedModel', modelId);
    isDropdownOpen = false;
    searchTerm = '';
    modelSelectorFocused = false;
  }

  function handleSearchFocus() {
    isDropdownOpen = true;
    modelSelectorFocused = true;
    // Clear searchTerm when focusing to start fresh
    searchTerm = '';
  }

  function handleSearchBlur() {
    // Small delay to allow click events to register
    setTimeout(() => {
      modelSelectorFocused = false;
      isDropdownOpen = false;
      // Reset search input to show selected model
      searchTerm = '';
    }, 200);
  }

  function handleClickOutside(event: MouseEvent) {
    const target = event.target as HTMLElement;
    if (!target.closest('.model-selector')) {
      isDropdownOpen = false;
    }
  }

  function searchModelMatch(modelId: string, modelName: string, searchQuery: string): boolean {
    // If empty search, show all
    if (!searchQuery.trim()) return true;

    // Convert everything to lowercase for case-insensitive matching
    const modelIdLower = modelId.toLowerCase();
    const modelNameLower = modelName.toLowerCase();
    const searchTerms = searchQuery.toLowerCase().split(/\s+/).filter(term => term.length > 0);

    // Check if all search terms are found in either the ID or name
    return searchTerms.every(term => 
      modelIdLower.includes(term) || modelNameLower.includes(term)
    );
  }

  $: {
    // Filter models whenever searchTerm or availableModels changes
    filteredModels = Object.entries(availableModels).reduce((acc, [id, model]) => {
      if (searchModelMatch(id, model.name, searchTerm)) {
        acc[id] = model;
      }
      return acc;
    }, {} as Record<string, OpenRouterModel>);
  }

  // Show the selected model name in the input
  $: selectedModelName = availableModels[selectedModel]?.name || selectedModel;

  function loadChat(chat: Chat) {
    currentChatId = chat.ID;
    // Update selected model to match the chat's model
    if (chat.ModelName && availableModels[chat.ModelName]) {
        selectedModel = chat.ModelName;
        localStorage.setItem('selectedModel', chat.ModelName);
    }
    messages = chat.Messages.map(msg => ({
        Role: msg.Role,
        Content: msg.Content,
        ID: msg.ID,
        ModelName: msg.ModelName || chat.ModelName, // Include ModelName from message or chat
        Starred: msg.Starred
    } as NewMessage));
    showChatHistory = false;
  }

  function formatDate(dateString: string) {
    return new Date(dateString).toLocaleString();
  }

  function startNewChat() {
    messages = [];
    currentChatId = null;
    showChatHistory = false;
  }

  // Helper function to format pricing
  function formatPrice(price: string): string {
    const num = parseFloat(price);
    if (num === 0) return 'Free';
    if (num < 0.00001) {
      return `$${(num * 1000000).toFixed(2)}/M tokens`;
    }
    return `$${num.toFixed(6)}`;
  }

  // Helper function to format modality
  function formatModality(modality: string): string {
    return modality.replace('->', ' → ');
  }

  async function toggleChatStar(chatId: number) {
    try {
        const response = await fetch(`http://localhost:8088/api/chat/${chatId}/star`, {
            method: 'POST',
        });
        
        if (response.ok) {
            const data = await response.json();
            // Update the chat's starred status in the previousChats array
            previousChats = previousChats.map(chat => 
                chat.ID === chatId ? { ...chat, Starred: data.starred } : chat
            );
        }
    } catch (error) {
        console.error('Error toggling chat star:', error);
    }
  }

  async function toggleMessageStar(messageId: number) {
    try {
        const response = await fetch(`http://localhost:8088/api/message/${messageId}/star`, {
            method: 'POST',
        });
        
        if (response.ok) {
            const data = await response.json();
            // Update the message's starred status in the messages array
            messages = messages.map(msg => 
                'ID' in msg && msg.ID === messageId ? { ...msg, Starred: data.starred } : msg
            );
        }
    } catch (error) {
        console.error('Error toggling message star:', error);
    }
  }
</script>

<svelte:window on:click={handleClickOutside}/>

<div class="chat-interface">
  <div class="sidebar" class:hidden={!showChatHistory}>
    <div class="sidebar-header">
      <h3>Chat History</h3>
      <div class="sidebar-buttons">
        <button 
          class="new-chat-button" 
          on:click={startNewChat}
        >
          + New Chat
        </button>
        <button 
          class="history-button" 
          on:click={() => showChatHistory = !showChatHistory}
        >
          {showChatHistory ? '←' : '→'}
        </button>
      </div>
    </div>
    <div class="chat-history">
      {#each [...previousChats].sort((a, b) => new Date(b.CreatedAt).getTime() - new Date(a.CreatedAt).getTime()) as chat}
        <div 
          class="chat-history-item" 
          class:starred={chat.Starred}
          on:click={() => loadChat(chat)}
          on:keydown={(e) => e.key === 'Enter' && loadChat(chat)}
          role="button"
          tabindex="0"
        >
          <div class="chat-preview">
            <div class="chat-header">
              <span class="chat-date">{formatDate(chat.CreatedAt)}</span>
              <button 
                class="star-button" 
                aria-label={chat.Starred ? "Unstar chat" : "Star chat"}
                on:click|stopPropagation={() => toggleChatStar(chat.ID)}
                title={chat.Starred ? "Unstar chat" : "Star chat"}
              >
                <svg class="star-icon" class:filled={chat.Starred} viewBox="0 0 24 24">
                  <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
                </svg>
              </button>
            </div>
            {#if chat.ModelName}
              <span class="chat-model">{availableModels[chat.ModelName]?.name || chat.ModelName}</span>
            {/if}
            <span class="chat-snippet">
              {chat.Messages?.[0]?.Content?.slice(0, 50) || 'Empty chat'}...
            </span>
          </div>
        </div>
      {/each}
    </div>
  </div>

  <div class="main-content">
    <div class="header">
      <button 
        class="history-button" 
        on:click={() => showChatHistory = !showChatHistory}
      >
        {showChatHistory ? '←' : '→'}
      </button>
      <div class="model-section">
        <div class="model-selector">
          <label for="model-search">Model:</label>
          <div class="dropdown-container">
            <input
              id="model-search"
              type="text"
              placeholder="Search models..."
              value={modelSelectorFocused ? searchTerm : selectedModelName}
              on:input={(e) => searchTerm = e.currentTarget.value}
              bind:this={searchInput}
              on:focus={handleSearchFocus}
              on:blur={handleSearchBlur}
              class="search-input"
              class:focused={modelSelectorFocused}
            />
            {#if isDropdownOpen}
              <div class="dropdown-list">
                {#each Object.entries(filteredModels) as [id, model]}
                  <div
                    class="dropdown-item"
                    class:selected={id === selectedModel}
                    on:mousedown={() => handleModelSelect(id)}
                    role="button"
                    tabindex="0"
                  >
                    <div class="model-info">
                      <div class="model-name-container">
                        <span class="model-name">{model.name}</span>
                        {#if model.description}
                          <span class="info-icon" title={model.description}>
                            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                              <circle cx="12" cy="12" r="10"></circle>
                              <line x1="12" y1="16" x2="12" y2="12"></line>
                              <line x1="12" y1="8" x2="12.01" y2="8"></line>
                            </svg>
                          </span>
                        {/if}
                      </div>
                      <div class="model-pricing">
                        <span class="price-tag">Input: {formatPrice(model.pricing.prompt)}</span>
                        <span class="price-tag">Output: {formatPrice(model.pricing.completion)}</span>
                      </div>
                    </div>
                    {#if id === selectedModel}
                      <span class="checkmark">✓</span>
                    {/if}
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        </div>
        {#if selectedModel && availableModels[selectedModel]}
          <div class="model-details">
            <div class="model-header">
              {#if availableModels[selectedModel].description}
                <span class="info-icon standalone" title={availableModels[selectedModel].description}>
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <circle cx="12" cy="12" r="10"></circle>
                    <line x1="12" y1="16" x2="12" y2="12"></line>
                    <line x1="12" y1="8" x2="12.01" y2="8"></line>
                  </svg>
                </span>
              {/if}
            </div>
            <div class="model-architecture">
              <span class="detail-tag">
                <span class="detail-label">Modality:</span>
                {formatModality(availableModels[selectedModel].architecture.modality)}
              </span>
              <span class="detail-tag">
                <span class="detail-label">Tokenizer:</span>
                {availableModels[selectedModel].architecture.tokenizer}
              </span>
              {#if availableModels[selectedModel].architecture.instruct_type}
                <span class="detail-tag">
                  <span class="detail-label">Instruct:</span>
                  {availableModels[selectedModel].architecture.instruct_type}
                </span>
              {/if}
            </div>
            <div class="model-pricing-display">
              <span class="detail-tag">
                <span class="detail-label">Input:</span>
                {formatPrice(availableModels[selectedModel].pricing.prompt)}
              </span>
              <span class="detail-tag">
                <span class="detail-label">Output:</span>
                {formatPrice(availableModels[selectedModel].pricing.completion)}
              </span>
            </div>
          </div>
        {/if}
      </div>
    </div>

    <div class="messages" bind:this={messagesContainer} on:scroll={handleScroll}>
      {#each messages as message}
        <div class="message-section">
          <div class="message-container">
            <ChatMessage 
                {message} 
                {availableModels} 
                onToggleStar={toggleMessageStar} 
            />
          </div>
        </div>
      {/each}
    </div>

    {#if !autoScroll && messages.length > 0}
      <button
        class="scroll-to-bottom-button"
        on:click={scrollToBottom}
      >
        Scroll to bottom
      </button>
    {/if}

    <form on:submit|preventDefault={handleSubmit}>
      <div class="input-container">
        <input
          type="text"
          bind:value={userInput}
          placeholder="Type your message..."
          disabled={isLoading}
        />
        <button type="submit" disabled={isLoading || !userInput.trim()}>
          Send
        </button>
      </div>
    </form>
  </div>
</div>

<style>
  .chat-interface {
    display: flex;
    height: 100vh;
    background-color: #1f1f1f;
    overflow: hidden;
  }

  .sidebar {
    width: 300px;
    border-right: 1px solid #333;
    background-color: #2a2a2a;
    display: flex;
    flex-direction: column;
    transition: width 0.3s ease;
  }

  .sidebar.hidden {
    width: 0;
    overflow: hidden;
  }

  .sidebar-header {
    padding: 0.75rem;
    border-bottom: 1px solid #333;
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 0.5rem;
  }

  .sidebar-header h3 {
    margin: 0;
    color: #e1e1e1;
    font-size: 1rem;
    white-space: nowrap;
  }

  .main-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
    overflow: hidden;
  }

  .chat-history {
    flex: 1;
    overflow-y: auto;
  }

  .chat-history-item {
    padding: 1rem;
    border-bottom: 1px solid #333;
    cursor: pointer;
  }

  .chat-history-item:hover {
    background-color: #3a3a3a;
  }

  .chat-preview {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .chat-date {
    font-size: 0.8rem;
    color: #888;
  }

  .chat-snippet {
    color: #e1e1e1;
    font-size: 0.9rem;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .messages {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    padding-bottom: 100px; /* Add padding to prevent overlap with fixed form */
  }

  .message-section {
    width: 100%;
    border-bottom: 1px solid #333;
    background-color: #1f1f1f;
    flex-shrink: 0; /* Add this to prevent shrinking */
  }

  .message-section:nth-child(even) {
    background-color: #2a2a2a;
  }

  .message-container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 2rem;
    width: 100%;
  }

  .header {
    padding: 1rem;
    border-bottom: 1px solid #333;
    display: flex;
    align-items: flex-start;
    gap: 1rem;
    flex-shrink: 0;
  }

  .input-container {
    padding: 1.5rem 2rem;
    border-top: 2px solid #333;
    background-color: #2a2a2a;
    display: flex;
    gap: 0.5rem;
    width: 100%;
    box-shadow: 0 -4px 6px rgba(0, 0, 0, 0.1);
  }

  .model-section {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    flex: 1;
  }

  .model-selector {
    padding: 1rem;
    border-bottom: 1px solid #333;
    color: #e1e1e1;
    flex-shrink: 0;
    min-width: 300px;
    max-width: 500px;
    flex-grow: 1;
  }

  input {
    flex-grow: 1;
    padding: 0.75rem 1rem;
    margin-right: 0.5rem;
    border: 1px solid #ccc;
    border-radius: 4px;
    font-size: 0.95rem;
  }

  button {
    padding: 0.5rem 1rem;
    background-color: #646cff;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
  }

  button:disabled {
    background-color: #ccc;
    cursor: not-allowed;
  }

  .dropdown-container {
    position: relative;
    width: 100%;
  }

  .search-input {
    width: 100%;
    padding: 0.75rem 1rem;
    border-radius: 4px;
    border: 1px solid #333;
    background-color: #2a2a2a;
    color: #e1e1e1;
    cursor: pointer;
    transition: all 0.2s ease;
    font-size: 0.95rem;
  }

  .search-input:hover {
    border-color: #646cff;
  }

  .search-input.focused {
    border-color: #646cff;
    box-shadow: 0 0 0 2px rgba(100, 108, 255, 0.2);
  }

  .dropdown-list {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    max-height: 300px;
    overflow-y: auto;
    background-color: #2a2a2a;
    border: 1px solid #444;
    border-radius: 4px;
    margin-top: 4px;
    z-index: 1000;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
    width: 100%;
  }

  .dropdown-item {
    padding: 0.75rem 1rem;
    cursor: pointer;
    color: #e1e1e1;
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    transition: background-color 0.2s ease;
    gap: 1rem;
  }

  .dropdown-item:hover {
    background-color: #3a3a3a;
  }

  .dropdown-item.selected {
    background-color: rgba(100, 108, 255, 0.1);
  }

  .model-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .model-pricing {
    display: flex;
    gap: 1rem;
    font-size: 0.8rem;
    color: #888;
  }

  .price-tag {
    background-color: rgba(100, 108, 255, 0.1);
    padding: 0.2rem 0.4rem;
    border-radius: 4px;
    white-space: nowrap;
  }

  .model-name {
    font-weight: 500;
    margin-bottom: 0.25rem;
  }

  .checkmark {
    color: #646cff;
    font-weight: bold;
  }

  /* Add smooth scrollbar for the dropdown */
  .dropdown-list {
    scrollbar-width: thin;
    scrollbar-color: #646cff #2a2a2a;
  }

  .dropdown-list::-webkit-scrollbar {
    width: 8px;
  }

  .dropdown-list::-webkit-scrollbar-track {
    background: #2a2a2a;
  }

  .dropdown-list::-webkit-scrollbar-thumb {
    background-color: #444;
    border-radius: 4px;
  }

  .dropdown-list::-webkit-scrollbar-thumb:hover {
    background-color: #646cff;
  }

  .history-button {
    padding: 0.5rem;
    min-width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .history-button:hover {
    background-color: #3a3a3a;
  }

  .sidebar-buttons {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .new-chat-button {
    padding: 0.5rem 0.75rem;
    background-color: #646cff;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.8rem;
    white-space: nowrap;
    min-width: fit-content;
  }

  .new-chat-button:hover {
    background-color: #747bff;
  }

  .model-details {
    display: flex;
    gap: 1rem;
    padding: 0.5rem;
    background: rgba(100, 108, 255, 0.05);
    border-radius: 4px;
    font-size: 0.9rem;
    align-items: center;
    flex-wrap: wrap;
  }

  .model-architecture,
  .model-pricing-display {
    display: flex;
    gap: 0.75rem;
    flex-wrap: wrap;
  }

  .detail-tag {
    background: rgba(100, 108, 255, 0.1);
    padding: 0.3rem 0.6rem;
    border-radius: 4px;
    color: #e1e1e1;
    display: flex;
    align-items: center;
    gap: 0.3rem;
    font-size: 0.85rem;
  }

  .detail-label {
    color: #888;
    font-weight: 500;
  }

  .chat-model {
    font-size: 0.8rem;
    color: #646cff;
    background: rgba(100, 108, 255, 0.1);
    padding: 0.2rem 0.4rem;
    border-radius: 4px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  form {
    position: fixed;
    bottom: 0;
    left: 300px;
    right: 0;
    z-index: 100;
    background-color: #1f1f1f;
  }

  /* Update the sidebar hidden state */
  .sidebar.hidden + .main-content form {
    left: 0;
  }

  /* Remove any height restrictions on messages */
  :global(.chat-message) {
    max-height: none !important;
    overflow: visible !important;
  }

  .model-name-container {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .info-icon {
    cursor: help;
    color: #888;
    display: inline-flex;
    align-items: center;
    transition: color 0.2s;
  }

  .info-icon:hover {
    color: #646cff;
  }

  .model-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .model-title {
    font-weight: 500;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  /* Update info-icon style to work in both contexts */
  .info-icon {
    cursor: help;
    color: #888;
    display: inline-flex;
    align-items: center;
    transition: color 0.2s;
    margin-left: 0.25rem;
  }

  .info-icon:hover {
    color: #646cff;
  }

  .info-icon.standalone {
    padding: 0.3rem;
    background: rgba(100, 108, 255, 0.1);
    border-radius: 4px;
    margin: 0;
  }

  .info-icon.standalone:hover {
    background: rgba(100, 108, 255, 0.2);
  }

  .chat-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .star-button {
    background: none;
    border: none;
    padding: 4px;
    cursor: pointer;
    color: #888;
    transition: color 0.2s;
  }

  .star-button:hover {
    color: #646cff;
  }

  .star-icon {
    width: 16px;
    height: 16px;
    fill: none;
    stroke: currentColor;
    stroke-width: 2;
    transition: fill 0.2s;
  }

  .star-icon.filled {
    fill: #646cff;
    stroke: #646cff;
  }

  .chat-history-item.starred {
    background-color: rgba(100, 108, 255, 0.1);
  }

  .scroll-to-bottom-button {
    padding: 0.5rem;
    background-color: #646cff;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    position: fixed;
    bottom: 100px;
    right: 20px;
    z-index: 100;
  }

  .scroll-to-bottom-button:hover {
    background-color: #747bff;
  }

  .scroll-to-bottom {
    position: fixed;
    bottom: 80px; /* Positioned above the fixed input container */
    right: 20px;
    padding: 0.5rem 1rem;
    background-color: #646cff;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    z-index: 110;
  }
</style> 