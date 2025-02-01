<script lang="ts">
  import ChatInterface from './lib/ChatInterface.svelte'
</script>

<main>
  <div class="chat-container">
    <ChatInterface />
  </div>
</main>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
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

  h1 {
    text-align: center;
    color: #e1e1e1;
    margin-bottom: 2rem;
  }
</style>
