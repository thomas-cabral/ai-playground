<script lang="ts">
  import { onMount, afterUpdate, tick } from 'svelte';
  import ChatMessage from './ChatMessage.svelte';
  import type { Message, NewMessage, Chat, OpenRouterModel, TokenUsage } from './types';

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
  let webSearchEnabled = false;

  let messagesContainer: HTMLDivElement;

  let autoScroll = true;

  // Update pagination variables
  let currentPage = 1;
  let hasMore = true;
  let isFetchingMore = false;
  let isInitialLoad = true;

  let editingMessageId: number | null = null;
  let editedContent = '';

  // Add these variables
  let forkHistory: { id: number; parentId: number | null; messageContent: string }[] = [];
  let currentForkId: number | null = null;

  // Add this type definition at the top with other types
  interface Fork {
    messageId: number;
    forkId: number;
    messageContent: string;
    createdAt: string;
  }

  // Update the messageForksMap type
  let messageForksMap: Map<number, Fork[]> = new Map();

  // Update the parent chat type
  let parentChat: { 
    id: number; 
    messageContent: string; 
    createdAt: string;
  } | null = null;

  // Single onMount function to handle all initialization
  onMount(async () => {
    const savedModel = localStorage.getItem('selectedModel');
    const urlParams = new URLSearchParams(window.location.search);
    const chatId = urlParams.get('chat');
    
    try {
      // 1. First fetch models
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

      // Set default model
      if (savedModel && availableModels[savedModel]) {
        selectedModel = savedModel;
      } else {
        selectedModel = Object.keys(availableModels).find(id => 
          id === 'anthropic/claude-3.5-haiku'
        ) || Object.keys(availableModels)[0];
      }

      filteredModels = { ...availableModels };

      // 2. Initial chat history fetch
      await fetchChatHistory(1, false);

      // 3. Set up intersection observer after initial fetch
      const observer = new IntersectionObserver(
        async (entries) => {
          const trigger = entries[0];
          if (trigger.isIntersecting && hasMore && !isFetchingMore) {
            await fetchChatHistory(currentPage + 1, true);
          }
        },
        {
          root: null,
          rootMargin: '100px',
          threshold: 0.1,
        }
      );

      if (loadMoreTrigger) {
        observer.observe(loadMoreTrigger);
      }

      // 4. If there's a chat ID in the URL, load it
      if (chatId) {
        await loadChatById(chatId);
      }

      return () => {
        if (loadMoreTrigger) {
          observer.unobserve(loadMoreTrigger);
        }
      };

    } catch (error) {
      console.error('Error in initialization:', error);
    }
  });

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

  function updateUrl(chatId: string | null) {
    const url = chatId ? `?chat=${chatId}` : window.location.pathname;
    window.history.replaceState({}, '', url);
  }

  // Update the loadForksForChat function to add logging
  async function loadForksForChat(chatId: number) {
    try {
      console.log('Loading forks for chat:', chatId);
      const response = await fetch(`http://localhost:8088/api/chat/${chatId}/forks`);
      if (response.ok) {
        const forks = await response.json();
        console.log('Received forks:', forks);
        messageForksMap.clear();
        
        // For each fork, get the message content only if we have a valid messageId
        for (const fork of forks) {
          console.log('Processing fork:', fork);
          if (!fork.messageId) {
            console.log('Skipping fork due to missing messageId:', fork);
            continue;
          }
          
          const existingForks = messageForksMap.get(fork.messageId) || [];
          existingForks.push({
            messageId: fork.messageId,
            forkId: fork.forkId,
            messageContent: fork.messageContent || '', 
            createdAt: fork.createdAt || new Date().toISOString()
          });
          messageForksMap.set(fork.messageId, existingForks);
        }
        
        console.log('Updated messageForksMap:', messageForksMap);
        // Force Svelte to update by creating a new Map
        messageForksMap = new Map(messageForksMap);
      }
    } catch (error) {
      console.error('Error loading forks:', error);
      messageForksMap = new Map();
    }
  }

  // Update loadChatById to load forks
  async function loadChatById(chatId: string) {
    try {
      const response = await fetch(`http://localhost:8088/api/chat/${chatId}`);
      if (response.ok) {
        const chat = await response.json();
        currentChatId = chat.id;
        messages = chat.messages.map(msg => ({
          role: msg.role,
          content: msg.content,
          id: msg.id,
          modelName: msg.modelName || chat.modelName,
          starred: msg.starred,
          tokenUsage: msg.tokenUsage,
        }));
        
        // Load parent info if this is a fork
        if (chat.parentId && chat.forkMessageId) {
          const parentResponse = await fetch(`http://localhost:8088/api/chat/${chat.parentId}/fork-message/${chat.forkMessageId}`);
          if (parentResponse.ok) {
            const parentData = await parentResponse.json();
            parentChat = {
              id: parentData.chatId,
              messageContent: parentData.messageContent,
              createdAt: parentData.createdAt
            };
          }
        } else {
          parentChat = null;
        }

        if (chat.modelName && availableModels[chat.modelName]) {
          selectedModel = chat.modelName;
          localStorage.setItem('selectedModel', chat.modelName);
        }

        // Load forks for the current chat
        await loadForksForChat(chat.id);
      } else {
        console.error('Chat not found');
        updateUrl(null);
      }
    } catch (error) {
      console.error('Error loading chat:', error);
      updateUrl(null);
    }
  }

  async function* streamResponse(reader: ReadableStreamDefaultReader<Uint8Array>) {
    let lastChunk = '';
    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        // Process any remaining data in lastChunk
        if (lastChunk) {
          try {
            const data = JSON.parse(lastChunk);
            if (data.usage) {
              // Return token usage data
              yield {
                type: 'usage',
                usage: {
                  promptTokens: data.usage.prompt_tokens,
                  completionTokens: data.usage.completion_tokens,
                  totalTokens: data.usage.total_tokens
                }
              };
            }
          } catch (e) {
            console.error('Error parsing final chunk:', e);
          }
        }
        break;
      }

      const text = new TextDecoder().decode(value);
      const lines = text.split('\n').filter(line => line.trim() !== '');

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          try {
            const data = JSON.parse(line.slice(6));
            if (data.choices?.[0]?.delta?.content) {
              yield {
                type: 'content',
                content: data.choices[0].delta.content
              };
            } else if (data.usage) {
              yield {
                type: 'usage',
                usage: {
                  promptTokens: data.usage.prompt_tokens,
                  completionTokens: data.usage.completion_tokens,
                  totalTokens: data.usage.total_tokens
                }
              };
            }
          } catch (e) {
            console.error('Error parsing JSON:', e);
          }
        }
      }
      lastChunk = lines[lines.length - 1];
    }
  }

  async function updateChatHistory() {
    try {
      const historyResponse = await fetch('http://localhost:8088/api/chat');
      if (historyResponse.ok) {
        const data = await historyResponse.json();
        previousChats = data.map((chat: any) => ({
          id: chat.id,
          messages: chat.messages.map((msg: Message) => ({
            id: msg.id,
            chatId: msg.chatId,
            role: msg.role,
            content: msg.content,
            createdAt: msg.createdAt,
            updatedAt: msg.updatedAt,
            deletedAt: msg.deletedAt,
            modelName: msg.modelName,
            starred: msg.starred,
            tokenUsage: ('promptTokens' in msg && 'completionTokens' in msg && 'totalTokens' in msg)
              ? {
                  promptTokens: msg.promptTokens,
                  completionTokens: msg.completionTokens,
                  totalTokens: msg.totalTokens
                }
              : undefined
          })),
          createdAt: chat.createdAt,
          updatedAt: chat.updatedAt,
          deletedAt: chat.deletedAt,
          modelName: chat.modelName,
          starred: chat.starred
        }));
      }
    } catch (error) {
      console.error('Error updating chat history:', error);
    }
  }

  async function handleSubmit() {
    if (!userInput.trim()) return;

    const newUserMessage = { 
        role: 'user', 
        content: userInput,
        modelName: selectedModel
    };
    
    const maxRetries = 2;
    let retryCount = 0;
    let success = false;

    while (retryCount <= maxRetries && !success) {
        try {
            if (retryCount > 0) {
                console.log(`Retrying request (attempt ${retryCount + 1})`);
            }

            // If this is a new chat, create it first
            if (currentChatId === null) {
                try {
                    const response = await fetch('http://localhost:8088/api/chat/new', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify({
                            model: selectedModel
                        }),
                    });

                    if (!response.ok) throw new Error('Failed to create new chat');
                    const data = await response.json();
                    currentChatId = data.id;
                    updateUrl(currentChatId?.toString() || null);
                } catch (error) {
                    console.error('Error creating new chat:', error);
                    throw error;
                }
            }

            messages = [...messages, newUserMessage];
            const currentInput = userInput;
            userInput = '';
            isLoading = true;

            // Create new assistant message
            messages = [...messages, { 
                role: 'assistant', 
                content: '',
                modelName: selectedModel 
            } as NewMessage];

            // Create the request body with the chat_id
            const requestBody = {
                model: webSearchEnabled ? `${selectedModel}:online` : selectedModel,
                messages: messages.slice(0, -1).map(msg => ({
                    id: 'ID' in msg ? msg.id : undefined,
                    role: msg.role,
                    content: msg.content
                })),
                stream: true,
                chat_id: currentChatId  // Always include chat_id now
            };

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
            for await (const chunk of streamResponse(reader)) {
                if (chunk.type === 'content' && chunk.content.trim()) {
                    hasContent = true;
                    messages[messages.length - 1].content += chunk.content;
                    messages = messages;
                } else if (chunk.type === 'usage') {
                    messages[messages.length - 1].tokenUsage = chunk.usage;
                    messages = messages;
                }
            }

            if (!hasContent) {
                // Remove empty message and retry
                messages = messages.slice(0, -2); // Remove both user and assistant messages
                if (retryCount === maxRetries) {
                    messages = [...messages, newUserMessage, {
                        role: 'assistant',
                        content: 'Sorry, the model returned empty responses. Please try again later.',
                        modelName: selectedModel
                    }];
                }
                throw new Error('Empty response');
            }

            success = true;
            await updateChatHistory();

            // After successful message submission, if this is a new chat
            if (currentChatId === null && messages.length === 2) { // First user+assistant message pair
                const chatResponse = await fetch('http://localhost:8088/api/chat');
                if (chatResponse.ok) {
                    const chats = await chatResponse.json();
                    const newChat = chats[chats.length - 1];
                    if (newChat) {
                        currentChatId = newChat.id;
                        updateUrl(newChat.id.toString());
                    }
                }
            }

        } catch (error) {
            console.error(`Error (attempt ${retryCount + 1}):`, error);
            if (retryCount === maxRetries) {
                messages = messages.slice(0, -2); // Remove failed messages
                messages = [...messages, newUserMessage, {
                    role: 'assistant',
                    content: 'Sorry, there was an error processing your request after multiple attempts.',
                    modelName: selectedModel
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

  // Update loadChat to be simpler and not trigger any message sends
  function loadChat(chat: Chat) {
    currentChatId = chat.id;
    updateUrl(chat.id.toString());
    
    if (chat.modelName && availableModels[chat.modelName]) {
      selectedModel = chat.modelName;
      localStorage.setItem('selectedModel', chat.modelName);
    }
    
    messages = chat.messages.map(msg => ({
      role: msg.role,
      content: msg.content,
      id: msg.id,
      modelName: msg.modelName || chat.modelName,
      starred: msg.starred,
      tokenUsage: msg.tokenUsage,
    }));
    
    // Add this line to load forks when loading from history
    loadForksForChat(chat.id);
    
    showChatHistory = false;
  }

  function formatDate(dateString: string) {
    return new Date(dateString).toLocaleString();
  }

  function startNewChat() {
    messages = [];
    currentChatId = null;
    updateUrl(null);
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

  // Helper function to calculate cost from token usage and pricing
  function calculateCost(
    tokenUsage: TokenUsage,
    pricing: { prompt: string; completion: string }
  ): { promptCost: number; completionCost: number; totalCost: number } {
    const promptCost = tokenUsage.promptTokens * parseFloat(pricing.prompt);
    const completionCost = tokenUsage.completionTokens * parseFloat(pricing.completion);
    return {
      promptCost,
      completionCost,
      totalCost: promptCost + completionCost
    };
  }

  // Helper function to format cost as currency
  function formatCost(cost: number): string {
    return `$${cost.toFixed(6)}`;
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
                chat.id === chatId ? { ...chat, starred: data.starred } : chat
            );
            
            // Also update the current chat if it's the one being starred
            if (currentChatId === chatId) {
                messages = messages.map(msg => ({
                    ...msg,
                    starred: data.starred
                }));
            }
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
                'ID' in msg && msg.id === messageId ? { ...msg, starred: data.starred } : msg
            );
        }
    } catch (error) {
        console.error('Error toggling message star:', error);
    }
  }

  // Add this helper function to calculate total tokens for a chat
  function calculateChatTokens(chat: Chat): TokenUsage {
    return chat.messages.reduce((total, msg) => {
      if (msg.tokenUsage) {
        return {
          promptTokens: total.promptTokens + msg.tokenUsage.promptTokens,
          completionTokens: total.completionTokens + msg.tokenUsage.completionTokens,
          totalTokens: total.totalTokens + msg.tokenUsage.totalTokens
        };
      }
      return total;
    }, { promptTokens: 0, completionTokens: 0, totalTokens: 0 });
  }

  // Add window popstate event listener to handle browser back/forward
  onMount(() => {
    window.addEventListener('popstate', () => {
      const urlParams = new URLSearchParams(window.location.search);
      const chatId = urlParams.get('chat');
      if (chatId) {
        loadChatById(chatId);
      } else {
        startNewChat();
      }
    });
  });

  // Add this function to handle textarea auto-resize
  function autoResize(e: Event) {
    const textarea = e.target as HTMLTextAreaElement;
    // Reset height to auto to get the correct scrollHeight
    textarea.style.height = 'auto';
    // Set new height based on scrollHeight, capped at 6 lines
    const lineHeight = parseFloat(getComputedStyle(textarea).lineHeight);
    const maxHeight = lineHeight * 6; // 6 lines
    textarea.style.height = `${Math.min(textarea.scrollHeight, maxHeight)}px`;
  }

  // Add this function after the toggleChatStar function
  async function deleteChat(chatId: number) {
    if (!confirm('Are you sure you want to delete this chat?')) {
        return;
    }

    try {
        const response = await fetch(`http://localhost:8088/api/chat/${chatId}`, {
            method: 'DELETE',
        });
        
        if (response.ok) {
            // Remove the chat from the previousChats array
            previousChats = previousChats.filter(chat => chat.id !== chatId);
            
            // If we're currently viewing the deleted chat, start a new one
            if (currentChatId === chatId) {
                startNewChat();
            }
        }
    } catch (error) {
        console.error('Error deleting chat:', error);
    }
  }

  // Update the chat history fetch function
  async function fetchChatHistory(page: number = 1, append: boolean = false) {
    try {
      if (!append) {
        isLoading = true;
        previousChats = []; // Clear existing chats on initial load
      } else {
        if (isFetchingMore) return;
        isFetchingMore = true;
      }

      const response = await fetch(`http://localhost:8088/api/chat?page=${page}`);
      if (response.ok) {
        const data = await response.json();
        
        const formattedChats = data.chats.map((chat: any) => ({
          id: chat.id,
          messages: chat.messages.map((msg: any) => ({
            id: msg.id,
            chatId: msg.chatId,
            role: msg.role,
            content: msg.content,
            createdAt: msg.createdAt,
            updatedAt: msg.updatedAt,
            deletedAt: msg.deletedAt,
            modelName: msg.modelName,
            starred: msg.starred,
            tokenUsage: ('promptTokens' in msg && 'completionTokens' in msg && 'totalTokens' in msg)
              ? {
                  promptTokens: msg.promptTokens,
                  completionTokens: msg.completionTokens,
                  totalTokens: msg.totalTokens
                }
              : undefined
          })),
          createdAt: chat.createdAt,
          updatedAt: chat.updatedAt,
          deletedAt: chat.deletedAt,
          modelName: chat.modelName,
          starred: chat.starred
        }));

        if (append) {
          previousChats = [...previousChats, ...formattedChats];
        } else {
          previousChats = formattedChats;
        }

        hasMore = data.hasMore;
        currentPage = page;
      }
    } catch (error) {
      console.error('Error fetching chat history:', error);
    } finally {
      isLoading = false;
      isFetchingMore = false;
    }
  }

  // Update the intersection observer
  let historyContainer: HTMLElement;
  let loadMoreTrigger: HTMLElement;

  async function handleEditMessage(message: Message) {
    editingMessageId = message.id;
    editedContent = message.content;
  }

  // Update loadForkHistory to handle fork chain correctly
  async function loadForkHistory(chatId: number): Promise<Array<{
    id: number;
    parentId: number | null;
    messageContent: string;
  }>> {
    try {
      const response = await fetch(`http://localhost:8088/api/chat/${chatId}`);
      if (response.ok) {
        const chat: Chat = await response.json();
        const history: Array<{
          id: number;
          parentId: number | null;
          messageContent: string;
        }> = [];
        
        if (chat.parentId) {
          // Load parent history first
          const parentHistory = await loadForkHistory(chat.parentId);
          history.push(...parentHistory);
        }
        
        // Add current chat to history
        history.push({
          id: chat.id,
          parentId: chat.parentId || null,
          messageContent: chat.forkMessageID ? 
            chat.messages.find((m: Message) => m.id === chat.forkMessageID)?.content || '' : ''
        });
        
        return history;
      }
      return [];
    } catch (error) {
      console.error('Error loading fork history:', error);
      return [];
    }
  }

  // Update handleSaveEdit to properly handle forking
  async function handleSaveEdit() {
    if (!editingMessageId || !editedContent.trim() || isLoading) return;

    try {
      isLoading = true;
      
      // Create the fork
      const response = await fetch('http://localhost:8088/api/chat/fork', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          chatId: currentChatId,
          messageId: editingMessageId,
          newContent: editedContent,
        }),
      });

      if (!response.ok) throw new Error('Failed to create fork');

      const data = await response.json();
      const newChatId = data.id;

      // Reset edit state
      editingMessageId = null;
      editedContent = '';

      // Load forks for the current chat to update the UI
      await loadForksForChat(currentChatId ?? 0);

      // Navigate to the new forked chat
      updateUrl(newChatId.toString());
      await loadChatById(newChatId.toString());

      // Send the edited message in the new chat
      userInput = editedContent;
      await handleSubmit();

    } catch (error) {
      console.error('Error creating fork:', error);
    } finally {
      isLoading = false;
    }
  }

  function cancelEdit() {
    editingMessageId = null;
    editedContent = '';
  }
</script>

<svelte:window on:click={handleClickOutside}/>

<div class="chat-interface">
  <div class="sidebar" class:hidden={!showChatHistory}>
    <div class="sidebar-header">
      <h3>Chat History</h3>
    </div>
    <div class="chat-history" bind:this={historyContainer}>
      {#if isLoading && !isFetchingMore}
        <div class="loading-state">
          <div class="loading-spinner"></div>
        </div>
      {/if}
      
      {#each previousChats as chat}
        <div 
          class="chat-history-item" 
          class:starred={chat.starred}
          on:click={() => loadChat(chat)}
          on:keydown={(e) => e.key === 'Enter' && loadChat(chat)}
          role="button"
          tabindex="0"
        >
          <div class="chat-preview">
            <div class="chat-header">
              <span class="chat-date">{formatDate(chat.createdAt)}</span>
              <div class="chat-actions">
                <button 
                  class="star-button" 
                  aria-label={chat.starred ? "Unstar chat" : "Star chat"}
                  on:click|stopPropagation={() => toggleChatStar(chat.id)}
                  title={chat.starred ? "Unstar chat" : "Star chat"}
                >
                  <svg class="star-icon" class:filled={chat.starred} viewBox="0 0 24 24">
                    <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
                  </svg>
                </button>
                <button 
                  class="delete-button" 
                  aria-label="Delete chat"
                  on:click|stopPropagation={() => deleteChat(chat.id)}
                  title="Delete chat"
                >
                  <svg viewBox="0 0 24 24" width="16" height="16">
                    <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
                  </svg>
                </button>
              </div>
            </div>
            {#if chat.modelName}
              <span class="chat-model">{availableModels[chat.modelName]?.name || chat.modelName}</span>
            {/if}
            <span class="chat-snippet">
              {#if chat.messages && chat.messages.length > 0}
                {chat.messages[0].content?.slice(0, 50)}...
              {:else}
                Empty chat...
              {/if}
            </span>
            {#if chat.messages && chat.messages.length > 0}
              <div class="chat-tokens">
                {#if chat.messages[0].tokenUsage}
                  {@const tokens = calculateChatTokens(chat)}
                  <span title="Total tokens used in chat">
                    🔤 {tokens.totalTokens.toLocaleString()}
                  </span>
                  {#if availableModels[chat.modelName]}
                    {@const cost = calculateCost(tokens, availableModels[chat.modelName].pricing)}
                    <span title="Total cost">
                      💲 {formatCost(cost.totalCost)}
                    </span>
                  {/if}
                {/if}
              </div>
            {/if}
          </div>
        </div>
      {/each}
      
      {#if hasMore}
        <div 
          class="load-more-trigger" 
          bind:this={loadMoreTrigger}
        >
          {#if isFetchingMore}
            <div class="loading-spinner"></div>
          {/if}
        </div>
      {/if}
    </div>
  </div>

  <div class="main-content">
    <div class="header">
      <div class="header-controls">
        <button 
          class="history-button" 
          on:click={() => showChatHistory = !showChatHistory}
        >
          {showChatHistory ? '←' : '→'}
        </button>
        <button 
          class="new-chat-button" 
          on:click={startNewChat}
        >
          + New Chat
        </button>
      </div>
      <div class="model-section">
        <div class="model-selector">
          <label for="model-search">Model:</label>
          <div class="model-controls">
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
            <button 
              class="web-search-toggle"
              class:enabled={webSearchEnabled}
              on:click={() => webSearchEnabled = !webSearchEnabled}
              aria-label={webSearchEnabled ? "Disable web search" : "Enable web search"}
              title={webSearchEnabled ? "Disable web search" : "Enable web search"}
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="10"></circle>
                <line x1="2" y1="12" x2="22" y2="12"></line>
                <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"></path>
              </svg>
            </button>
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

    <!-- Add parent chat navigation -->
    {#if parentChat}
      <div class="parent-chat-nav">
        <div class="parent-header">
          <span class="parent-label">Forked from chat #{parentChat.id}</span>
        </div>
        <div 
          class="parent-content"
          on:click={() => loadChatById(parentChat.id.toString())}
        >
          <div class="parent-message">
            <div class="parent-info">
              <div class="parent-meta">
                <span class="parent-icon">↑</span>
                <span class="parent-time">{formatDate(parentChat.createdAt)}</span>
              </div>
              <div class="parent-text">
                <span class="parent-description">Original message:</span>
                {parentChat.messageContent}
              </div>
            </div>
          </div>
        </div>
      </div>
    {/if}

    <div class="messages" bind:this={messagesContainer} on:scroll={handleScroll}>
      {#if messages.length === 0}
        <div class="empty-state">
          <div class="empty-state-content">
            <h2>Welcome to the AI Playground</h2>
            <p>Start a conversation by typing your message below.</p>
            <div class="empty-state-tips">
              <h3>Tips:</h3>
              <ul>
                <li>Choose a model from the dropdown above</li>
                <li>Toggle web search for up-to-date information</li>
                <li>Use Shift+Enter for multi-line messages</li>
                <li>Access chat history using the sidebar</li>
              </ul>
            </div>
          </div>
        </div>
      {/if}
      {#each messages as message}
        <div class="message-section">
          <div class="message-container">
            {#if message.id === editingMessageId}
              <div class="edit-container">
                <textarea
                  bind:value={editedContent}
                  rows="4"
                  class="edit-textarea"
                ></textarea>
                <div class="edit-actions">
                  <button on:click={handleSaveEdit}>Save & Fork</button>
                  <button on:click={cancelEdit} class="cancel-button">Cancel</button>
                </div>
              </div>
            {:else}
              {#if message.id}
                {@const messageForks = messageForksMap.get(message.id) || []}
                {@const debug = console.log('Message forks for', message.id, ':', messageForks)}
              {/if}
              <ChatMessage 
                {message} 
                {availableModels}
                onToggleStar={toggleMessageStar}
                onEdit={handleEditMessage}
                forks={messageForksMap.get(message.id ?? -1) || []}
                showForks={true}
                onShowForks={() => {
                  console.log('Show forks clicked for message:', message.id);
                  if (message.id) {
                    loadForksForChat(currentChatId ?? 0);
                  }
                }}
              />
            {/if}
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
        <textarea
          bind:value={userInput}
          placeholder="Type your message... (Shift+Enter for new line)"
          disabled={isLoading}
          on:keydown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault();
              if (userInput.trim()) handleSubmit();
            }
          }}
          on:input={autoResize}
          rows="1"
        ></textarea>
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
    --sidebar-visible: 1;
  }

  .sidebar.hidden {
    width: 0;
    overflow: hidden;
    --sidebar-visible: 0;
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
    height: 100%;
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
    cursor: pointer;
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
    padding: 1rem;
  }

  .message-section {
    width: 100%;
    border-bottom: 1px solid #333;
    background-color: #1f1f1f;
    flex-shrink: 0;
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
    background-color: #2a2a2a;
  }

  .header-controls {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    height: 32px;
  }

  .input-container {
    padding: 1.5rem 2rem 2rem;
    border-top: 2px solid #333;
    background-color: #2a2a2a;
    display: flex;
    gap: 0.5rem;
    width: 100%;
    box-shadow: 0 -4px 6px rgba(0, 0, 0, 0.1);
    align-items: flex-start;
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

  textarea {
    flex-grow: 1;
    padding: 0.75rem 1rem;
    margin-right: 0.5rem;
    border: 1px solid #ccc;
    border-radius: 4px;
    font-size: 0.95rem;
    min-height: 32px;
    max-height: calc(1.2em * 6 + 1.5rem);
    line-height: 1.2;
    box-sizing: border-box;
    resize: none;
    overflow-y: auto;
    font-family: inherit;
    background-color: #fff;
    transition: height 0.1s ease;
  }

  textarea:disabled {
    background-color: #eee;
    cursor: not-allowed;
  }

  /* Add smooth scrollbar for textarea */
  textarea {
    scrollbar-width: thin;
    scrollbar-color: #646cff #ffffff;
  }

  textarea::-webkit-scrollbar {
    width: 8px;
  }

  textarea::-webkit-scrollbar-track {
    background: #ffffff;
  }

  textarea::-webkit-scrollbar-thumb {
    background-color: #646cff;
    border-radius: 4px;
    border: 2px solid #ffffff;
  }

  /* Update the button styles */
  .input-container button {
    padding: 0 1rem;
    height: calc(32px + 0.5rem); /* Reduced from 1.5rem to 0.5rem for better proportion */
    align-self: flex-start; /* Changed from stretch to flex-start */
    margin-top: 1px; /* Slight adjustment to align with textarea */
    display: flex;
    align-items: center;
    justify-content: center;
  }

  button {
    padding: 0 0.75rem;
    background-color: #646cff;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.8rem;
    white-space: nowrap;
    min-width: fit-content;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  button:disabled {
    background-color: #ccc;
    cursor: not-allowed;
  }

  .model-controls {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    width: 100%;
  }

  .dropdown-container {
    position: relative;
    flex: 1;
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
    padding: 0;
    min-width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .history-button:hover {
    background-color: #3a3a3a;
  }

  .new-chat-button {
    padding: 0 0.75rem;
    background-color: #646cff;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.8rem;
    white-space: nowrap;
    min-width: fit-content;
    height: 32px;
    display: flex;
    align-items: center;
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
    background-color: #1f1f1f;
    margin-top: auto;
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

  .web-search-toggle {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0.75rem;
    background-color: #2a2a2a;
    color: #888;
    border: 1px solid #444;
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.2s ease;
    flex-shrink: 0;
  }

  .web-search-toggle:hover {
    background-color: #3a3a3a;
    border-color: #646cff;
  }

  .web-search-toggle.enabled {
    background-color: rgba(100, 108, 255, 0.1);
    color: #646cff;
    border-color: #646cff;
  }

  .web-search-toggle svg {
    transition: transform 0.2s ease;
  }

  .web-search-toggle.enabled svg {
    transform: scale(1.1);
  }

  .chat-tokens {
    font-size: 0.8rem;
    color: #888;
    margin-top: 0.25rem;
  }

  .chat-tokens span {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    background: rgba(100, 108, 255, 0.1);
    padding: 0.2rem 0.4rem;
    border-radius: 4px;
    cursor: help;
  }

  .chat-actions {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .delete-button {
    background: none;
    border: none;
    padding: 4px;
    cursor: pointer;
    color: #888;
    transition: color 0.2s;
  }

  .delete-button:hover {
    color: #ff4444;
  }

  .delete-button svg {
    fill: currentColor;
  }

  .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 400px;
    padding: 2rem;
  }

  .empty-state-content {
    text-align: center;
    max-width: 600px;
    color: #e1e1e1;
  }

  .empty-state-content h2 {
    margin: 0 0 1rem 0;
    font-size: 1.5rem;
    color: #646cff;
  }

  .empty-state-content p {
    margin: 0 0 2rem 0;
    color: #888;
  }

  .empty-state-tips {
    text-align: left;
    background: rgba(100, 108, 255, 0.1);
    padding: 1.5rem;
    border-radius: 8px;
  }

  .empty-state-tips h3 {
    margin: 0 0 1rem 0;
    font-size: 1rem;
    color: #646cff;
  }

  .empty-state-tips ul {
    margin: 0;
    padding-left: 1.5rem;
    color: #888;
  }

  .empty-state-tips li {
    margin-bottom: 0.5rem;
  }

  .empty-state-tips li:last-child {
    margin-bottom: 0;
  }

  .load-more-trigger {
    height: 50px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .loading-spinner {
    width: 20px;
    height: 20px;
    border: 2px solid #646cff;
    border-top-color: transparent;
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  /* Add loading state styles */
  .loading-state {
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 2rem;
  }

  /* Add styles for message count */
  .message-count {
    font-size: 0.8rem;
    color: #888;
    background: rgba(100, 108, 255, 0.1);
    padding: 0.2rem 0.4rem;
    border-radius: 4px;
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  .edit-container {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    padding: 1rem;
  }

  .edit-textarea {
    width: 100%;
    min-height: 100px;
    padding: 0.75rem;
    border: 1px solid #444;
    border-radius: 4px;
    background-color: #2a2a2a;
    color: #e1e1e1;
    font-family: inherit;
    resize: vertical;
  }

  .edit-actions {
    display: flex;
    gap: 0.5rem;
    justify-content: flex-end;
  }

  .cancel-button {
    background-color: #444;
  }

  .cancel-button:hover {
    background-color: #555;
  }

  .parent-chat-nav {
    padding: 0.75rem;
    border-bottom: 1px solid #333;
    background-color: #2a2a2a;
  }

  .parent-header {
    margin-bottom: 0.5rem;
  }

  .parent-label {
    font-size: 0.8rem;
    color: #888;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .parent-content {
    display: flex;
    flex-direction: column;
    padding: 0.75rem;
    background: rgba(100, 108, 255, 0.1);
    border-radius: 4px;
    color: #e1e1e1;
    cursor: pointer;
    transition: background-color 0.2s;
  }

  .parent-content:hover {
    background-color: rgba(100, 108, 255, 0.2);
  }

  .parent-message {
    display: flex;
    gap: 0.75rem;
  }

  .parent-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .parent-meta {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .parent-icon {
    color: #646cff;
    font-size: 1rem;
  }

  .parent-time {
    color: #888;
    font-size: 0.8rem;
  }

  .parent-description {
    color: #888;
    font-size: 0.8rem;
    display: block;
    margin-bottom: 0.25rem;
  }

  .parent-text {
    font-size: 0.9rem;
    color: #e1e1e1;
    line-height: 1.5;
  }
</style> 