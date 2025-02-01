<script lang="ts">
  import { marked } from 'marked';
  import { onMount } from 'svelte';

  export let message: {
    Role: string;
    Content: string;
  };

  // Configure marked for safe HTML
  marked.setOptions({
    breaks: true,  // Convert \n to <br>
    gfm: true,     // GitHub Flavored Markdown
    headerIds: false,
    mangle: false
  });

  $: formattedContent = marked(message.Content);
</script>

<div class="message {message.Role.toLowerCase()}">
  <div class="avatar">
    {message.Role === 'user' ? '👤' : '🤖'}
  </div>
  <div class="content-wrapper">
    <div class="content">
      {@html formattedContent}
    </div>
  </div>
</div>

<style>
  .message {
    display: flex;
    margin-bottom: 1rem;
    gap: 0.5rem;
    color: #e1e1e1;
    align-self: flex-start;
    width: 100%;
    max-height: 50vh; /* Limit maximum height */
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
    max-height: 60vh; /* Match message max-height */
    overflow-y: auto; /* Enable vertical scrolling */
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
</style> 