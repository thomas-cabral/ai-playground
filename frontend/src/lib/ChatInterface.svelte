<script lang="ts">
  import { onMount } from 'svelte';
  import ChatMessage from './ChatMessage.svelte';

  type Message = {
    ID: number;
    ChatID: number;
    Role: string;
    Content: string;
    CreatedAt: string;
    UpdatedAt: string;
    DeletedAt: string | null;
  };

  // Add this new type for messages being composed
  type NewMessage = {
    Role: string;
    Content: string;
    ID?: number;  // Add optional ID field
  };

  type Chat = {
    ID: number;
    Messages: Message[];
    CreatedAt: string;
    UpdatedAt: string;
    DeletedAt: string | null;
  };

  const LOCAL_MODELS = {
    'anthropic/claude-3.5-sonnet': 'Anthropic Claude 3 Sonnet',
    'anthropic/claude-3.5-haiku': 'Claude 3.5 Haiku',
  };

  let availableModels = { ...LOCAL_MODELS };
  let messages: (Message | NewMessage)[] = [];
  let userInput = '';
  let isLoading = false;
  let selectedModel = 'anthropic/claude-3.5-haiku';
  let searchTerm = '';
  let filteredModels: Record<string, string> = availableModels;
  let isDropdownOpen = false;
  let searchInput: HTMLInputElement;
  let previousChats: Chat[] = [];
  let showChatHistory = false;
  let currentChatId: number | null = null;

  onMount(async () => {
    // Get saved model from localStorage
    const savedModel = localStorage.getItem('selectedModel');
    
    try {
      const response = await fetch('https://openrouter.ai/api/v1/models');
      
      if (!response.ok) throw new Error('Failed to fetch models');
      
      const data = await response.json();
      const openRouterModels: Record<string, string> = {};
      
      data.data.forEach((model: { id: string; name: string }) => {
          openRouterModels[`${model.id}`] = model.name;
      });

      // Trigger Svelte reactivity by reassigning the variable
      availableModels = {
        ...LOCAL_MODELS,
        ...openRouterModels
      };

      // After models are loaded, check if saved model exists in available models
      if (savedModel && availableModels[savedModel]) {
        selectedModel = savedModel;
      }
    } catch (error) {
      console.error('Error fetching OpenRouter models:', error);
      // Ensure we at least have local models if the fetch fails
      availableModels = { ...LOCAL_MODELS };
      
      // Still check saved model against LOCAL_MODELS
      if (savedModel && LOCAL_MODELS[savedModel]) {
        selectedModel = savedModel;
      }
    }

    // Add this to fetch chat history
    try {
      const response = await fetch('http://localhost:8088/api/history');
      if (response.ok) {
        previousChats = await response.json();
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

  async function handleSubmit() {
    if (!userInput.trim()) return;

    // Create new message without ID since it's new
    const newUserMessage = { Role: 'user', Content: userInput };
    messages = [...messages, newUserMessage];
    const currentInput = userInput;
    userInput = '';
    isLoading = true;

    try {
        // First, if this is a new chat, create it
        if (!currentChatId) {
            const createChatResponse = await fetch('http://localhost:8088/api/chat/new', {
                method: 'POST',
            });
            
            if (!createChatResponse.ok) {
                throw new Error('Failed to create new chat');
            }
            
            const chatData = await createChatResponse.json();
            currentChatId = chatData.id;
        }

        // Now send the message
        const response = await fetch('http://localhost:8088/api/chat', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                model: selectedModel,
                chat_id: currentChatId,
                messages: messages.map(msg => ({
                    id: 'ID' in msg ? msg.ID : undefined,
                    role: msg.Role,
                    content: msg.Content
                })),
                stream: true,
            }),
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const reader = response.body?.getReader();
        if (!reader) throw new Error('No reader available');

        // Create new assistant message
        messages = [...messages, { Role: 'assistant', Content: '' } as NewMessage];

        for await (const content of streamResponse(reader)) {
            messages[messages.length - 1].Content += content;
            messages = messages; // Trigger Svelte reactivity
        }
    } catch (error) {
        console.error('Error:', error);
        messages = [...messages, { 
            Role: 'assistant', 
            Content: 'Sorry, there was an error processing your request.' 
        }];
    } finally {
        isLoading = false;
    }
  }

  function handleModelSelect(modelId: string) {
    selectedModel = modelId;
    // Save to localStorage when model is selected
    localStorage.setItem('selectedModel', modelId);
    isDropdownOpen = false;
    searchTerm = '';
  }

  function handleSearchFocus() {
    isDropdownOpen = true;
  }

  function handleClickOutside(event: MouseEvent) {
    const target = event.target as HTMLElement;
    if (!target.closest('.model-selector')) {
      isDropdownOpen = false;
    }
  }

  $: {
    // Filter models whenever searchTerm or availableModels changes
    filteredModels = Object.entries(availableModels).reduce((acc, [id, name]) => {
      const searchLower = searchTerm.toLowerCase();
      if (name.toLowerCase().includes(searchLower) || id.toLowerCase().includes(searchLower)) {
        acc[id] = name;
      }
      return acc;
    }, {} as Record<string, string>);
  }

  // Show the selected model name in the input
  $: selectedModelName = availableModels[selectedModel as keyof typeof availableModels] || selectedModel;

  function loadChat(chat: Chat) {
    currentChatId = chat.ID;
    messages = chat.Messages.map(msg => ({
      Role: msg.Role,
      Content: msg.Content,
      ID: msg.ID
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
          on:click={() => loadChat(chat)}
          on:keydown={(e) => e.key === 'Enter' && loadChat(chat)}
          role="button"
          tabindex="0"
        >
          <div class="chat-preview">
            <span class="chat-date">{formatDate(chat.CreatedAt)}</span>
            <span class="chat-snippet">
              {chat.Messages[0]?.Content?.slice(0, 50) || 'Empty chat'}...
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
      <div class="model-selector">
        <label for="model-search">Model:</label>
        <div class="dropdown-container">
          <input
            id="model-search"
            type="text"
            placeholder="Search models..."
            value={isDropdownOpen ? searchTerm : selectedModelName}
            on:input={(e) => searchTerm = e.currentTarget.value}
            bind:this={searchInput}
            on:focus={handleSearchFocus}
            class="search-input"
          />
          {#if isDropdownOpen}
            <div class="dropdown-list">
              {#each Object.entries(filteredModels) as [id, name]}
                <div
                  class="dropdown-item"
                  class:selected={id === selectedModel}
                  on:click={() => handleModelSelect(id)}
                  on:keydown={(e) => e.key === 'Enter' && handleModelSelect(id)}
                  role="button"
                  tabindex="0"
                >
                  {name}
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    </div>

    <div class="messages">
      {#each messages as message}
        <ChatMessage {message} />
      {/each}
    </div>

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
    min-width: 0; /* Prevents flex items from overflowing */
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
    padding: 1rem;
  }

  .header {
    padding: 1rem;
    border-bottom: 1px solid #333;
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .input-container {
    padding: 1rem;
    border-top: 1px solid #333;
    background-color: #2a2a2a;
    display: flex;
    gap: 0.5rem;
  }

  .model-selector {
    padding: 1rem;
    border-bottom: 1px solid #333;
    color: #e1e1e1;
    flex-shrink: 0; /* Prevent model selector from shrinking */
  }

  input {
    flex-grow: 1;
    padding: 0.5rem;
    margin-right: 0.5rem;
    border: 1px solid #ccc;
    border-radius: 4px;
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
    padding: 0.5rem;
    border-radius: 4px;
    border: 1px solid #333;
    background-color: #2a2a2a;
    color: #e1e1e1;
    cursor: pointer;
  }

  .dropdown-list {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    max-height: 200px;
    overflow-y: auto;
    background-color: #2a2a2a;
    border: 1px solid #333;
    border-radius: 4px;
    margin-top: 4px;
    z-index: 1000;
  }

  .dropdown-item {
    padding: 0.5rem;
    cursor: pointer;
    color: #e1e1e1;
  }

  .dropdown-item:hover {
    background-color: #3a3a3a;
  }

  .dropdown-item.selected {
    background-color: #464646;
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
</style> 