<script lang="ts">
  import {
    CastIcon,
    DownloadIcon,
    GitBranchIcon,
    HardDriveIcon,
    ImageIcon,
    LockIcon,
    PackageIcon,
    RefreshCwIcon,
    Share2Icon,
  } from "svelte-feather-icons";
  import { browser } from '$app/environment';

  import logotypeDark from "$lib/assets/logotype-dark.svg";
  import landingGraphic from "$lib/assets/landing-graphic.svg";
  import landingBackground from "$lib/assets/landing-background.svg";
  import TeaserVideo from "$lib/ui/TeaserVideo.svelte";
  import CopyableCode from "$lib/ui/CopyableCode.svelte";
  import DownloadLink from "$lib/ui/DownloadLink.svelte";

  let installationEl: HTMLDivElement;
  
  // Get the current domain for dynamic URLs
  $: baseUrl = browser ? window.location.origin : '';
  $: getScriptUrl = baseUrl ? `${baseUrl}/get` : '/get';

  const socials = [
    {
      title: "🤖  GitHub",
      href: "https://github.com/ovidiuvio/SSHXtend",
    },
    {
      title: "📦  Releases",
      href: "https://github.com/ovidiuvio/SSHXtend/releases",
    },
    {
      title: "📚  Wiki",
      href: "https://github.com/ovidiuvio/SSHXtend/wiki",
    },
    {
      title: "🌎  Original sshx",
      href: "https://github.com/ekzhang/sshx",
    },
  ];

  function scrollToInstallation() {
    installationEl.scrollIntoView({ behavior: "smooth" });
  }
</script>

<main
  class="max-w-screen-xl mx-auto px-4 md:px-8 lg:px-16 text-zinc-100 overflow-x-hidden"
>
  <header class="mt-6 mb-4 sm:my-8 md:my-12">
    <div class="flex items-center gap-3">
      <h1 class="text-4xl font-bold">SSHXtend</h1>
      <span class="bg-pink-700 text-white text-sm px-3 py-1 rounded-full">Enhanced Fork</span>
    </div>
  </header>
  <h1
    class="font-medium text-3xl sm:text-4xl md:text-5xl max-w-[30ch] py-2 mb-6 md:mb-0 sm:tracking-tight leading-[1.15]"
  >
    A secure web-based,
    <span class="title-gradient">AI-powered collaborative</span> terminal
  </h1>

  <div class="relative">
    <div
      class="absolute scale-150 md:scale-100 md:left-[180px] md:top-[-200px] md:w-[1000px] -z-10"
    >
      <img class="select-none" src={landingBackground} alt="" />
    </div>
    <div class="md:absolute md:left-[500px] md:w-[1000px]">
      <img
        class="mt-5 mb-8 w-[720px]"
        width={813}
        height={623}
        src={landingGraphic}
        alt="two terminal windows running sshx and three live cursors"
      />
    </div>
  </div>

  <section class="my-12 space-y-6 sm:text-lg md:max-w-[460px] text-zinc-400">
    <p>
      <code class="name">SSHXtend</code> is an enhanced fork of sshx that lets you share
      your terminal with anyone by link, on a
      <b>multiplayer infinite canvas</b>.
    </p>
    <p>
      It has <b>real-time collaboration</b>, with remote cursors and chat.
      Plus <b>AI assistance</b>, <b>enterprise dashboards</b>, and <b>100+ themes</b>.
      It's also <b>fast</b> and <b>end-to-end encrypted</b>, with a lightweight
      server written in Rust.
    </p>
    <p>
      Run <code class="name">sshx</code> instantly without installation, or install it permanently.
      Use it for teaching, debugging, cloud access, or enterprise monitoring.
    </p>
  </section>

  <div class="pb-12 md:pb-36">
    <button
      class="bg-pink-700 hover:bg-pink-600 active:ring-4 active:ring-pink-500/50 text-lg font-medium px-8 py-2 rounded-full"
      on:click={scrollToInstallation}
    >
      Get Started
    </button>
  </div>

  <div class="mt-8 md:mt-32 grid md:grid-cols-2 xl:grid-cols-3 gap-4 md:gap-6">
    <div class="feature-block">
      <div class="feature-icon">
        <CastIcon size="14" />
      </div>
      <h3>AI-Powered Assistant</h3>
      <p>Built-in AI support with Gemini and OpenRouter integration.</p>
    </div>
    <div class="feature-block">
      <div class="feature-icon">
        <LockIcon size="14" />
      </div>
      <h3>End-to-end encrypted</h3>
      <p>Send data securely; the server never sees what you're typing.</p>
    </div>
    <div class="feature-block">
      <div class="feature-icon">
        <HardDriveIcon size="14" />
      </div>
      <h3>Enterprise Dashboards</h3>
      <p>Monitor and manage multiple sessions with real-time analytics.</p>
    </div>
    <div class="feature-block">
      <div class="feature-icon">
        <ImageIcon size="14" />
      </div>
      <h3>Infinite canvas</h3>
      <p>Move and resize multiple terminals at once, in any arrangement.</p>
    </div>
    <div class="feature-block">
      <div class="feature-icon">
        <RefreshCwIcon size="14" />
      </div>
      <h3>Live presence</h3>
      <p>See other people's names and cursors within the app.</p>
    </div>
    <div class="feature-block">
      <div class="feature-icon">
        <Share2Icon size="14" />
      </div>
      <h3>Ultra-fast mesh networking</h3>
      <p>
        Connect from anywhere to the nearest distributed peer in a global
        network.
      </p>
    </div>
  </div>

  <div class="my-48 hidden md:block">
    <TeaserVideo />
  </div>

  <h2
    bind:this={installationEl}
    class="mt-32 mb-12 font-medium text-3xl sm:text-4xl md:text-center scroll-mt-16"
  >
    Installation
  </h2>

  <section class="installation-section">
    <h3 class="text-xl sm:text-lg">
      <DownloadIcon size="20" class="text-zinc-400 inline-block mr-1 mb-0.5" />
      macOS / Linux
    </h3>
    <div class="text-sm text-zinc-400 md:text-base md:pt-0.5">
      <p class="mb-3">Quick start - install and run immediately:</p>
      <CopyableCode value={`curl -sSf ${getScriptUrl} | sh -s run`} />
      
      <p class="mt-6 mb-3">Or install to /usr/local/bin:</p>
      <CopyableCode value={`curl -sSf ${getScriptUrl} | sh`} />
      
      <p class="mt-6 mb-3">Run instantly with original sshx.io server (no install):</p>
      <CopyableCode value={`curl -sSf ${getScriptUrl} | sh -s run -- --server https://sshx.io`} />
      
      <div class="mt-8 p-4 bg-zinc-800/50 rounded-lg border border-zinc-700">
        <h4 class="text-zinc-100 font-medium mb-3">Dashboard Commands</h4>
        <p class="text-sm text-zinc-400 mb-3">Launch and create a monitoring dashboard without installation:</p>
        <div class="mb-3">
          <CopyableCode value={`curl -sSf ${getScriptUrl} | sh -s run -- --dashboard`} />
        </div>
        <p class="text-sm text-zinc-400 mb-3">Or after installation, create a dashboard:</p>
        <div class="mb-3">
          <CopyableCode value="sshx --dashboard" />
        </div>
        <p class="text-sm text-zinc-400 mb-3">Join an existing dashboard with a key:</p>
        <div class="mb-3">
          <CopyableCode value="sshx --dashboard <key>" />
        </div>
        <p class="text-sm text-zinc-400 mb-3">Dashboards provide real-time monitoring of all sessions, user activity, and system metrics in a centralized web interface.</p>
        <div class="mt-3 p-3 bg-yellow-900/20 border border-yellow-700/50 rounded">
          <p class="text-xs text-yellow-400">
            <strong>⚠️ Security Notice:</strong> Anyone with the dashboard key can view all sessions in that dashboard. 
            Only share dashboard keys with trusted team members.
          </p>
        </div>
      </div>

      <p class="mt-8 mb-3">Or, download the binary directly from GitHub:</p>
      <div class="flex flex-wrap gap-2 mb-2">
        <DownloadLink
          href="https://github.com/ovidiuvio/SSHXtend/releases/latest/download/sshxtend-aarch64-apple-darwin"
          >macOS ARM64 (Apple Silicon)</DownloadLink
        >
        <DownloadLink
          href="https://github.com/ovidiuvio/SSHXtend/releases/latest/download/sshxtend-x86_64-apple-darwin"
          >macOS x86-64 (Intel)</DownloadLink
        >
      </div>
      <div class="flex flex-wrap gap-2 mb-2">
        <DownloadLink
          href="https://github.com/ovidiuvio/SSHXtend/releases/latest/download/sshxtend-aarch64-unknown-linux-musl"
          >Linux ARM64</DownloadLink
        >
        <DownloadLink
          href="https://github.com/ovidiuvio/SSHXtend/releases/latest/download/sshxtend-x86_64-unknown-linux-musl"
          >Linux x86-64</DownloadLink
        >
      </div>
      <div class="flex flex-wrap gap-2">
        <DownloadLink
          href="https://github.com/ovidiuvio/SSHXtend/releases/latest"
          >More architectures (MIPS, RISC-V, s390x)</DownloadLink
        >
      </div>
    </div>
  </section>

  <section class="installation-section">
    <h3 class="text-xl sm:text-lg">
      <DownloadIcon size="20" class="text-zinc-400 inline-block mr-1 mb-0.5" />
      Windows
    </h3>
    <div class="text-sm text-zinc-400 md:text-base md:pt-0.5">
      <p class="mb-3">Download the executable for your platform.</p>

      <div class="flex flex-wrap gap-2">
        <DownloadLink
          href="https://github.com/ovidiuvio/SSHXtend/releases/latest/download/sshxtend-x86_64-pc-windows-msvc.exe"
          >Windows x86-64</DownloadLink
        >
        <DownloadLink
          href="https://github.com/ovidiuvio/SSHXtend/releases/latest/download/sshxtend-aarch64-pc-windows-msvc.exe"
          >Windows ARM64</DownloadLink
        >
      </div>
    </div>
  </section>

  <section class="installation-section">
    <h3 class="text-xl sm:text-lg">
      <PackageIcon size="20" class="text-zinc-400 inline-block mr-1 mb-0.5" />
      Build from source
    </h3>
    <div class="text-sm text-zinc-400 md:text-base md:pt-0.5">
      <p class="mb-3">
        Ensure you have up-to-date versions of Rust and protoc. Clone the repo
        and build SSHXtend.
      </p>
      <CopyableCode value="git clone https://github.com/ovidiuvio/SSHXtend && cd SSHXtend && cargo build --release" />
    </div>
  </section>

  <section class="installation-section">
    <h3 class="text-xl sm:text-lg">
      <GitBranchIcon size="20" class="text-zinc-400 inline-block mr-1 mb-0.5" />
      CI/CD Integration
    </h3>
    <div class="text-sm text-zinc-400 md:text-base md:pt-0.5">
      <p class="mb-3">
        On GitHub Actions, GitLab CI/CD, or other CI providers, run this command.
        It pauses your workflow and starts a collaborative debugging session.
      </p>
      <CopyableCode value={`curl -sSf ${getScriptUrl} | sh -s run`} />
    </div>
  </section>

  <hr class="mt-32 mb-12" />

  <div class="grid sm:grid-cols-2 lg:grid-cols-4 gap-4 md:gap-6 mb-6">
    {#each socials as social}
      <a
        target="_blank"
        rel="noreferrer"
        href={social.href}
        class="border border-white/10 hover:border-white/30 transition-colors p-4 text-center text-lg font-medium rounded-lg"
      >
        {social.title}
      </a>
    {/each}
  </div>

  <p class="mb-12 text-center text-zinc-400">
    SSHXtend - Enhanced fork of <a href="https://github.com/ekzhang/sshx" class="text-pink-400 hover:underline">sshx</a> by Eric Zhang
  </p>
</main>

<style lang="postcss">
  b {
    @apply text-zinc-300 font-medium;
  }

  code.name {
    @apply text-[0.9em] text-zinc-100 border border-white/25 px-1 py-0.5 rounded;
  }

  hr {
    @apply mx-auto md:w-1/2 border-zinc-800;
  }

  .title-gradient {
    @apply text-transparent bg-clip-text bg-gradient-to-r from-fuchsia-400 to-blue-500;
  }

  .feature-block {
    @apply relative border rounded-lg border-transparent p-6 sm:p-8;
    background: #111111 padding-box;
  }

  .feature-block::before {
    content: "";
    @apply absolute inset-0;
    z-index: -1;
    margin: -1px;
    border-radius: inherit;
    background: linear-gradient(
      160deg,
      rgba(255, 255, 255, 0.36) 5%,
      rgba(255, 255, 255, 0.08) 25%,
      rgba(255, 255, 255, 0.24) 50%,
      rgba(255, 255, 255, 0.08) 75%,
      rgba(255, 255, 255, 0.28) 95%
    );
    opacity: 0.5;
    transition: opacity 200ms;
  }

  .feature-block:hover::before {
    opacity: 1;
  }

  .feature-block h3 {
    @apply font-medium mb-2;
  }

  .feature-block p {
    @apply text-zinc-400;
  }

  .feature-icon {
    @apply inline-block p-3 rounded-full mb-3 shadow-md border border-zinc-600;
  }

  .installation-section {
    @apply grid sm:grid-cols-[200px,1fr] gap-x-10 gap-y-4 max-w-4xl mx-auto sm:border-t sm:border-white/10 sm:py-6 lg:px-2;
    @apply mb-16 lg:mb-8;
  }
</style>