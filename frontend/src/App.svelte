<script lang="ts">
  import ChatInterface from './lib/ChatInterface.svelte'
</script>

<main>
  <div class="chat-container">
    <ChatInterface />
  </div>
</main>

<style>
  :global(html, body) {
    margin: 0;
    padding: 0;
    height: 100%;
    overflow: hidden;  /* Prevents scrolling on the viewport */
    box-sizing: border-box;
  }

  :global(*) {
    box-sizing: inherit;
  }

  main {
    min-height: 100vh;
    padding: 1rem;
    background-color: #1a1a1a;
    color: #e1e1e1;
  }
</style>
