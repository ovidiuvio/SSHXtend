<script lang="ts">
  import logotypeDark from '$lib/assets/logotype-dark.svg';
</script>

<svelte:head>
  <title>sshx - Secure Web-based Collaborative Terminal</title>
  <meta name="description" content="Share your terminal instantly with sshx" />
</svelte:head>

<main class="min-h-screen bg-theme-bg flex items-center justify-center">
  <div class="text-center space-y-6 max-w-2xl px-6">
    <img src={logotypeDark} alt="sshx" class="h-16 mx-auto" />
    
    <div class="space-y-4">
      <h1 class="text-3xl font-bold text-theme-fg">
        Secure Web-based Collaborative Terminal
      </h1>
      
      <p class="text-lg text-theme-fg-muted">
        Share your terminal instantly over the web with end-to-end encryption.
      </p>
    </div>

    <div class="bg-theme-bg-secondary rounded-lg p-6 space-y-4">
      <h2 class="text-lg font-semibold text-theme-fg">Quick Start</h2>
      
      <div class="text-left space-y-3">
        <div>
          <p class="text-sm text-theme-fg-muted mb-1">Share your terminal:</p>
          <code class="block bg-theme-bg px-3 py-2 rounded text-sm text-orange-600 dark:text-orange-400">
            curl -sSf https://sshx.io/get | sh && sshx
          </code>
        </div>
        
        <div>
          <p class="text-sm text-theme-fg-muted mb-1">Create a dashboard to monitor sessions:</p>
          <code class="block bg-theme-bg px-3 py-2 rounded text-sm text-orange-600 dark:text-orange-400">
            sshx --dashboard
          </code>
        </div>
        
        <div>
          <p class="text-sm text-theme-fg-muted mb-1">Join an existing dashboard:</p>
          <code class="block bg-theme-bg px-3 py-2 rounded text-sm text-orange-600 dark:text-orange-400">
            sshx --dashboard &lt;key&gt;
          </code>
        </div>
      </div>
    </div>

    <div class="text-sm text-theme-fg-muted">
      <p>
        Visit <a href="https://github.com/ekzhang/sshx" class="text-orange-600 dark:text-orange-400 hover:underline" target="_blank">GitHub</a> 
        for documentation and source code.
      </p>
    </div>
  </div>
</main>

<style>
  :global(body) {
    background-color: var(--theme-bg);
  }
</style>