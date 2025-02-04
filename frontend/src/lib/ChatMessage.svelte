<script lang="ts">
  import { marked, type MarkedOptions } from 'marked';
  import { onMount, afterUpdate } from 'svelte';
  import 'prismjs';
  import * as Prism from 'prismjs';
  
  // Import Prism theme
  import 'prismjs/themes/prism-tomorrow.css';
  
  // Import languages in dependency order
  import 'prismjs/components/prism-c';
  import 'prismjs/components/prism-cpp';
  import 'prismjs/components/prism-python';
  import 'prismjs/components/prism-typescript';
  import 'prismjs/components/prism-jsx';
  import 'prismjs/components/prism-tsx';
  import 'prismjs/components/prism-css';
  import 'prismjs/components/prism-scss';
  import 'prismjs/components/prism-json';
  import 'prismjs/components/prism-java';
  import 'prismjs/components/prism-go';
  import 'prismjs/components/prism-rust';
  import 'prismjs/components/prism-bash';
  import 'prismjs/components/prism-yaml';
  import 'prismjs/components/prism-sql';
  import 'prismjs/components/prism-graphql';
  import 'prismjs/components/prism-csharp';
  import type { Message, NewMessage } from './types';
  
  export let message: Message | NewMessage;
  export let availableModels: Record<string, any>; 
  export let onToggleStar: (messageId: number) => void;
  export let onEdit: (message: Message) => void;
  export let showForks: boolean = false;
  export let onShowForks: () => void;
  export let forks: { messageId: number; forkId: number; messageContent: string; createdAt: string }[] = [];

  let contentElement: HTMLElement;
  let formattedContent = ''; 

  // Add state for fork visibility
  let isForksVisible = false;

  // Function to toggle fork visibility
  function toggleForks() {
    isForksVisible = !isForksVisible;
    if (isForksVisible) {
      onShowForks();
    }
  }

  // Configure marked for safe HTML and syntax highlighting
  marked.setOptions({
    breaks: true,
    gfm: true,
    highlight: function(code: string, lang: string): string | Promise<string> {
      if (!lang) return code;
      
      lang = lang.toLowerCase();
      const langMap: { [key: string]: string } = {
        'js': 'javascript',
        'ts': 'typescript',
        'py': 'python',
        'yml': 'yaml',
        'shell': 'bash',
        'sh': 'bash',
        'jsx': 'jsx',
        'tsx': 'tsx',
        'scss': 'scss',
        'rust': 'rust',
        'go': 'go',
        'cs': 'csharp',
        'rb': 'ruby',
        'md': 'markdown'
      };
      
      const normalizedLang = langMap[lang] || lang;
      
      try {
        if (Prism.languages[normalizedLang]) {
          return Prism.highlight(code, Prism.languages[normalizedLang], normalizedLang);
        }
      } catch (e) {
        console.warn(`Failed to highlight ${normalizedLang}:`, e);
      }
      return code;
    }
  } as MarkedOptions);

  $: {
    Promise.resolve(marked(message.content)).then(content => {
      formattedContent = content;
    });
  }

  function highlightCode() {
    if (contentElement) {
      requestAnimationFrame(() => {
        const codeBlocks = contentElement.querySelectorAll('pre code');
        codeBlocks.forEach((block: Element) => {
          const className = block.className;
          const lang = className?.match(/language-(\w+)/)?.[1];
          if (block.textContent && lang && Prism.languages[lang]) {
            Prism.highlightElement(block);
          }
        });
      });
    }
  }

  afterUpdate(highlightCode);
  onMount(highlightCode);

  function getModelDisplayName(modelName: string): string {
    return availableModels[modelName]?.name || modelName;
  }

  // Helper function to calculate cost from token usage and pricing
  function calculateCost(
    tokenUsage: { promptTokens: number; completionTokens: number; totalTokens: number },
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

  function formatDate(date: string) {
    return new Date(date).toLocaleString();
  }
</script>

<div class="message {message.role.toLowerCase()} {message.starred ? 'starred' : ''}">
  <div class="avatar">
    {message.role === 'user' ? '👤' : '🤖'}
  </div>
  <div class="content-wrapper">
    <div class="message-header">
      <span class="role">{message.role}</span>
      {#if message.modelName}
        <span class="model-tag">
          {getModelDisplayName(message.modelName)}
        </span>
      {/if}
      {#if message.id}
        <button 
          class="star-button" 
          on:click={() => onToggleStar(message.id ?? 0)}
          title={message.starred ? "Unstar message" : "Star message"}
          aria-label={message.starred ? "Unstar message" : "Star message"}
        >
          <svg class="star-icon" class:filled={message.starred} viewBox="0 0 24 24">
            <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
          </svg>
        </button>
      {/if}
      {#if message.role === 'user'}
        <div class="message-actions">
          <div class="action-buttons">
            <button 
              class="edit-button" 
              on:click={() => onEdit(message)}
              title="Edit message"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
              </svg>
            </button>
            {#if forks.length > 0}
              <button 
                class="fork-button" 
                class:active={isForksVisible}
                on:click={toggleForks}
                title={isForksVisible ? "Hide forks" : "Show forks"}
              >
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M6 3v12"></path>
                  <circle cx="18" cy="6" r="3"></circle>
                  <circle cx="6" cy="18" r="3"></circle>
                  <path d="M18 9a9 9 0 0 1-9 9"></path>
                </svg>
                <span class="fork-count">{forks.length}</span>
              </button>
            {/if}
          </div>
        </div>
      {/if}
    </div>
    <div class="content" bind:this={contentElement}>
      {@html formattedContent}
      {#if message.role !== 'user' && message.tokenUsage && availableModels[message.modelName ?? '']}
        {@const cost = calculateCost(message.tokenUsage, availableModels[message.modelName ?? ''].pricing)}
        <div class="message-cost-details">
          <span title="Input cost">I: {formatCost(cost.promptCost)}</span>
          <span title="Output cost">O: {formatCost(cost.completionCost)}</span>
          <span title="Total cost">Total: {formatCost(cost.totalCost)}</span>
        </div>
      {/if}
    </div>
  </div>

  {#if isForksVisible && forks.length > 0}
    <div class="forks-container">
      <div class="forks-header">
        <span class="forks-title">Forks ({forks.length})</span>
      </div>
      <div class="forks-list">
        {#each forks as fork}
          <div class="fork-item" on:click={() => window.location.href = `?chat=${fork.forkId}`}>
            <div class="fork-content">
              <div class="fork-preview">{fork.messageContent.slice(0, 100)}...</div>
              <div class="fork-meta">
                <span class="fork-date">{formatDate(fork.createdAt)}</span>
              </div>
            </div>
            <div class="fork-arrow">→</div>
          </div>
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  .message {
    display: flex;
    margin-bottom: 1rem;
    gap: 0.5rem;
    color: #e1e1e1;
    width: 100%;
  }

  .avatar {
    font-size: 1.5rem;
    min-width: 2rem;
  }

  .content-wrapper {
    flex: 1;
    max-width: 95%;
    position: relative;
  }

  .content {
    padding: 0.5rem 1rem;
    border-radius: 8px;
    width: 100%;
  }

  .content :global(pre) {
    background: #2a2a2a;
    padding: 1rem;
    border-radius: 4px;
    overflow-x: auto;
    border: 1px solid #3a3a3a;
    width: 100%;
    text-align: left;
    margin: 1rem 0;
    max-width: 100%;
  }

  .content :global(pre code) {
    text-align: left;
    display: block;
    font-family: 'Fira Code', 'Consolas', monospace;
    font-size: 0.9em;
    line-height: 1.5;
  }

  /* Optional: Style inline code differently from code blocks */
  .content :global(:not(pre) > code) {
    background: #2a2a2a;
    padding: 0.2em 0.4em;
    border-radius: 3px;
    font-size: 0.9em;
    border: 1px solid #3a3a3a;
  }

  .content :global(code) {
    font-family: 'Fira Code', 'Consolas', monospace;
    color: #e1e1e1;
    background: rgba(255, 255, 255, 0.1);
    padding: 0.2rem 0.4rem;
    border-radius: 3px;
  }

  .content :global(p) {
    margin: 0.5rem 0;
    line-height: 1.5;
    text-align: left;
  }

  .content :global(ul), .content :global(ol) {
    margin: 0.5rem 0;
    padding-left: 1.5rem;
    text-align: left;
  }

  .content :global(li) {
    text-align: left;
  }

  .content :global(h1), 
  .content :global(h2), 
  .content :global(h3), 
  .content :global(h4), 
  .content :global(h5), 
  .content :global(h6) {
    text-align: left;
    margin: 1rem 0 0.5rem 0;
  }

  .user .content {
    background-color: #4a4fff;
    color: #ffffff;
  }

  .user .content :global(code) {
    background: rgba(255, 255, 255, 0.15);
    color: #ffffff;
  }

  .assistant .content {
    background-color: #2a2a2a;
    color: #e1e1e1;
  }

  .assistant .content :global(pre code) {
    text-align: left;
    display: block;
  }

  .assistant .content :global(a) {
    color: #7c83ff;
  }

  .assistant .content :global(strong) {
    color: #ffffff;
  }

  .assistant .content :global(em) {
    color: #b8b8b8;
  }

  .assistant .content :global(pre)::-webkit-scrollbar {
    height: 8px;
    background-color: #2a2a2a;
  }

  .assistant .content :global(pre)::-webkit-scrollbar-thumb {
    background-color: #4a4a4a;
    border-radius: 4px;
  }

  .assistant .content :global(pre)::-webkit-scrollbar-track {
    background-color: #2a2a2a;
  }

  .content :global(table) {
    border-collapse: collapse;
    width: 100%;
    margin: 1rem 0;
  }

  .content :global(th),
  .content :global(td) {
    border: 1px solid #3a3a3a;
    padding: 0.5rem;
    text-align: left;
  }

  .content :global(th) {
    background-color: #2a2a2a;
  }

  /* Add scrollbar styling for the content */
  .content::-webkit-scrollbar {
    width: 8px;
    background-color: #2a2a2a;
  }

  .content::-webkit-scrollbar-thumb {
    background-color: #4a4a4a;
    border-radius: 4px;
  }

  .content::-webkit-scrollbar-track {
    background-color: #2a2a2a;
  }

  /* Add Prism theme styles */
  .content :global(pre code.language-javascript),
  .content :global(pre code.language-typescript),
  .content :global(pre code.language-css),
  .content :global(pre code.language-json),
  .content :global(pre code.language-bash),
  .content :global(pre code.language-go) {
    color: #f8f8f2;
  }

  .content :global(.token.comment),
  .content :global(.token.prolog),
  .content :global(.token.doctype),
  .content :global(.token.cdata) {
    color: #8292a2;
  }

  .content :global(.token.punctuation) {
    color: #f8f8f2;
  }

  .content :global(.token.property),
  .content :global(.token.tag),
  .content :global(.token.constant),
  .content :global(.token.symbol),
  .content :global(.token.deleted) {
    color: #ff79c6;
  }

  .content :global(.token.boolean),
  .content :global(.token.number) {
    color: #bd93f9;
  }

  .content :global(.token.selector),
  .content :global(.token.attr-name),
  .content :global(.token.string),
  .content :global(.token.char),
  .content :global(.token.builtin),
  .content :global(.token.inserted) {
    color: #50fa7b;
  }

  .content :global(.token.operator),
  .content :global(.token.entity),
  .content :global(.token.url),
  .content :global(.language-css .token.string),
  .content :global(.style .token.string),
  .content :global(.token.variable) {
    color: #f8f8f2;
  }

  .content :global(.token.atrule),
  .content :global(.token.attr-value),
  .content :global(.token.function),
  .content :global(.token.class-name) {
    color: #f1fa8c;
  }

  .content :global(.token.keyword) {
    color: #ff79c6;
  }

  .content :global(.token.regex),
  .content :global(.token.important) {
    color: #ffb86c;
  }

  .message-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
    flex-wrap: wrap;
  }

  .role {
    text-transform: capitalize;
    font-weight: 500;
    color: #646cff;
  }

  .model-tag {
    font-size: 0.8rem;
    color: #888;
    background: rgba(100, 108, 255, 0.1);
    padding: 0.2rem 0.4rem;
    border-radius: 4px;
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

  .message.starred {
    background-color: rgba(100, 108, 255, 0.05);
  }

  .token-usage {
    display: flex;
    gap: 1rem;
    margin-top: 0.5rem;
    font-size: 0.8rem;
    color: #888;
  }

  .token-usage span {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    background: rgba(100, 108, 255, 0.1);
    padding: 0.2rem 0.4rem;
    border-radius: 4px;
    cursor: help;
  }

  .message-cost-details {
    display: flex;
    gap: 1rem;
    margin-top: 0.5rem;
    font-size: 0.8rem;
    color: #888;
  }

  .message-actions {
    margin-left: auto; /* Push actions to the right */
  }

  .action-buttons {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .edit-button,
  .fork-button {
    background: none;
    border: none;
    padding: 4px;
    cursor: pointer;
    color: #888;
    transition: color 0.2s;
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  .edit-button:hover,
  .fork-button:hover {
    color: #646cff;
  }

  .fork-button.active {
    color: #646cff;
    background: rgba(100, 108, 255, 0.1);
    border-radius: 4px;
  }

  .fork-count {
    font-size: 0.8rem;
    background: rgba(100, 108, 255, 0.2);
    padding: 0.1rem 0.3rem;
    border-radius: 0.8rem;
    min-width: 1.2rem;
    text-align: center;
  }

  .forks-container {
    margin-top: 0.5rem;
    padding-top: 0.5rem;
    border-top: 1px solid #333;
  }
  
  .forks-header {
    margin-bottom: 0.5rem;
  }
  
  .forks-title {
    font-size: 0.9rem;
    color: #888;
  }
  
  .forks-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  
  .fork-item {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.75rem;
    background: rgba(100, 108, 255, 0.1);
    border-radius: 4px;
    cursor: pointer;
    transition: background-color 0.2s;
  }
  
  .fork-item:hover {
    background: rgba(100, 108, 255, 0.2);
  }
  
  .fork-content {
    flex: 1;
  }
  
  .fork-preview {
    font-size: 0.9rem;
    color: #e1e1e1;
    margin-bottom: 0.25rem;
  }
  
  .fork-meta {
    font-size: 0.8rem;
    color: #888;
  }
  
  .fork-arrow {
    color: #646cff;
    font-size: 1.2rem;
  }
</style> 