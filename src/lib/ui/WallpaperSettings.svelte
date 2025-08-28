<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { UploadIcon, XIcon, Trash2Icon, ImageIcon } from "svelte-feather-icons";
  import { wallpaperManager, type Wallpaper, type WallpaperFit } from "$lib/wallpaper";
  import { settings, updateSettings, type Settings } from "$lib/settings";
  import { makeToast } from "$lib/toast";

  const dispatch = createEventDispatcher();

  let fileInput: HTMLInputElement;
  let uploadProgress = false;
  let selectedWallpaper = $settings.wallpaperCurrent;
  let wallpaperFit = $settings.wallpaperFit;
  let wallpaperOpacity = $settings.wallpaperOpacity;
  let wallpaperEnabled = $settings.wallpaperEnabled;

  // Get all available wallpapers
  $: allWallpapers = wallpaperManager.getAllWallpapers();
  $: builtinWallpapers = wallpaperManager.getBuiltinWallpapers();
  $: customWallpapers = wallpaperManager.getCustomWallpapers();
  $: storageUsage = wallpaperManager.getStorageUsage();

  // Apply changes to settings
  $: {
    updateSettings({
      wallpaperEnabled,
      wallpaperCurrent: selectedWallpaper,
      wallpaperFit,
      wallpaperOpacity,
    });
  }

  function handleWallpaperSelect(wallpaper: Wallpaper) {
    selectedWallpaper = wallpaper.id;
  }

  function handleUploadClick() {
    fileInput?.click();
  }

  async function handleFileUpload(event: Event) {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    
    if (!file) return;

    uploadProgress = true;
    
    try {
      const newWallpaper = await wallpaperManager.addCustomWallpaper(file);
      if (newWallpaper) {
        selectedWallpaper = newWallpaper.id;
        makeToast({
          kind: "success",
          message: `Wallpaper "${newWallpaper.name}" added successfully`,
        });
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unknown error";
      makeToast({
        kind: "error",
        message: `Failed to upload wallpaper: ${message}`,
      });
    } finally {
      uploadProgress = false;
      if (input) input.value = "";
    }
  }

  function handleDeleteCustomWallpaper(wallpaper: Wallpaper) {
    if (wallpaperManager.removeCustomWallpaper(wallpaper.id)) {
      if (selectedWallpaper === wallpaper.id) {
        selectedWallpaper = "none";
      }
      makeToast({
        kind: "success",
        message: `Wallpaper "${wallpaper.name}" removed`,
      });
    }
  }

  function formatFileSize(bytes: number): string {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  }

  const fitOptions: { value: WallpaperFit; label: string; description: string }[] = [
    { value: "cover", label: "Cover", description: "Scale to fill entire area, may crop" },
    { value: "contain", label: "Contain", description: "Scale to fit entirely, may show empty space" },
    { value: "fill", label: "Fill", description: "Stretch to fill exactly, may distort" },
    { value: "tile", label: "Tile", description: "Repeat image as tiles" },
    { value: "center", label: "Center", description: "Show at original size, centered" },
  ];
</script>

<div class="space-y-6">
  <!-- Enable/Disable Toggle -->
  <div class="flex items-center justify-between">
    <div>
      <h3 class="text-lg font-medium text-theme-fg">Wallpapers</h3>
      <p class="text-sm text-theme-fg-secondary">
        Customize the desktop background shared by all terminals
      </p>
    </div>
    <label class="relative inline-flex items-center cursor-pointer">
      <input
        type="checkbox"
        class="sr-only peer"
        bind:checked={wallpaperEnabled}
      />
      <div class="w-11 h-6 bg-theme-bg-secondary rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-500"></div>
    </label>
  </div>

  {#if wallpaperEnabled}
    <!-- Wallpaper Grid -->
    <div class="space-y-4">
      <!-- Built-in Wallpapers -->
      <div>
        <h4 class="text-md font-medium text-theme-fg mb-3">Built-in Wallpapers</h4>
        <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
          {#each builtinWallpapers as wallpaper (wallpaper.id)}
            <button
              class="group relative aspect-video rounded-lg overflow-hidden border-2 transition-all hover:scale-105"
              class:border-blue-500={selectedWallpaper === wallpaper.id}
              class:border-theme-border={selectedWallpaper !== wallpaper.id}
              on:click={() => handleWallpaperSelect(wallpaper)}
            >
              {#if wallpaper.type === "none"}
                <div class="w-full h-full bg-theme-bg flex items-center justify-center">
                  <div class="text-center">
                    <div class="w-8 h-8 mx-auto mb-1 rounded bg-theme-bg-secondary flex items-center justify-center">
                      <div class="w-2 h-2 bg-theme-border rounded-full"></div>
                    </div>
                    <span class="text-xs text-theme-fg-secondary">Dots</span>
                  </div>
                </div>
              {:else}
                <img
                  src={wallpaper.thumbnail || wallpaper.url}
                  alt={wallpaper.name}
                  class="w-full h-full object-cover"
                />
              {/if}
              <div class="absolute inset-0 bg-black bg-opacity-0 group-hover:bg-opacity-20 transition-all">
                {#if selectedWallpaper === wallpaper.id}
                  <div class="absolute top-1 right-1 w-5 h-5 bg-blue-500 rounded-full flex items-center justify-center">
                    <svg class="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
                      <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"></path>
                    </svg>
                  </div>
                {/if}
              </div>
              <div class="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black to-transparent p-2">
                <span class="text-xs text-white font-medium">{wallpaper.name}</span>
              </div>
            </button>
          {/each}
        </div>
      </div>

      <!-- Custom Wallpapers -->
      <div>
        <div class="flex items-center justify-between mb-3">
          <h4 class="text-md font-medium text-theme-fg">Custom Wallpapers</h4>
          <button
            class="flex items-center gap-2 px-3 py-1.5 text-sm bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors disabled:opacity-50"
            on:click={handleUploadClick}
            disabled={uploadProgress}
          >
            {#if uploadProgress}
              <div class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
              Uploading...
            {:else}
              <UploadIcon size="16" />
              Add Image
            {/if}
          </button>
        </div>

        {#if customWallpapers.length > 0}
          <div class="grid grid-cols-2 sm:grid-cols-3 gap-3 mb-3">
            {#each customWallpapers as wallpaper (wallpaper.id)}
              <div class="group relative aspect-video rounded-lg overflow-hidden border-2 transition-all"
                   class:border-blue-500={selectedWallpaper === wallpaper.id}
                   class:border-theme-border={selectedWallpaper !== wallpaper.id}>
                <button
                  class="w-full h-full hover:scale-105 transition-transform"
                  on:click={() => handleWallpaperSelect(wallpaper)}
                >
                  <img
                    src={wallpaper.thumbnail || wallpaper.url}
                    alt={wallpaper.name}
                    class="w-full h-full object-cover"
                  />
                  <div class="absolute inset-0 bg-black bg-opacity-0 group-hover:bg-opacity-20 transition-all">
                    {#if selectedWallpaper === wallpaper.id}
                      <div class="absolute top-1 right-1 w-5 h-5 bg-blue-500 rounded-full flex items-center justify-center">
                        <svg class="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
                          <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"></path>
                        </svg>
                      </div>
                    {/if}
                  </div>
                </button>
                <button
                  class="absolute top-1 left-1 w-6 h-6 bg-red-500 hover:bg-red-600 text-white rounded-full flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
                  on:click={() => handleDeleteCustomWallpaper(wallpaper)}
                  title="Remove wallpaper"
                >
                  <XIcon size="14" />
                </button>
                <div class="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black to-transparent p-2">
                  <span class="text-xs text-white font-medium truncate">{wallpaper.name}</span>
                </div>
              </div>
            {/each}
          </div>

          <!-- Storage Usage -->
          <div class="text-xs text-theme-fg-secondary">
            Storage used: {formatFileSize(storageUsage.used)} ({storageUsage.percentage}%)
            {#if storageUsage.percentage > 80}
              <span class="text-yellow-500 ml-1">⚠️ Storage nearly full</span>
            {/if}
          </div>
        {:else}
          <div class="text-center py-8 text-theme-fg-secondary">
            <ImageIcon size="48" class="mx-auto mb-2 opacity-50" />
            <p class="text-sm">No custom wallpapers yet</p>
            <p class="text-xs">Upload images up to 5MB</p>
          </div>
        {/if}
      </div>

      <!-- Wallpaper Settings -->
      {#if selectedWallpaper !== "none"}
        <div class="space-y-4 pt-4 border-t border-theme-border">
          <h4 class="text-md font-medium text-theme-fg">Display Settings</h4>
          
          <!-- Fit Style -->
          <div>
            <label for="wallpaper-fit" class="block text-sm font-medium text-theme-fg mb-2">Fit</label>
            <select
              id="wallpaper-fit"
              bind:value={wallpaperFit}
              class="w-full px-3 py-2 bg-theme-bg-secondary border border-theme-border rounded-lg text-theme-fg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              {#each fitOptions as option}
                <option value={option.value}>{option.label} - {option.description}</option>
              {/each}
            </select>
          </div>

          <!-- Opacity -->
          <div>
            <label for="wallpaper-opacity" class="block text-sm font-medium text-theme-fg mb-2">
              Opacity: {Math.round(wallpaperOpacity * 100)}%
            </label>
            <input
              id="wallpaper-opacity"
              type="range"
              min="0.1"
              max="1.0"
              step="0.1"
              bind:value={wallpaperOpacity}
              class="w-full h-2 bg-theme-bg-secondary rounded-lg appearance-none cursor-pointer slider"
            />
          </div>
        </div>
      {/if}
    </div>
  {/if}
</div>

<!-- Hidden file input -->
<input
  type="file"
  bind:this={fileInput}
  on:change={handleFileUpload}
  accept="image/*"
  class="hidden"
/>

<style>
  .slider::-webkit-slider-thumb {
    appearance: none;
    height: 20px;
    width: 20px;
    border-radius: 50%;
    background: #3B82F6;
    cursor: pointer;
  }

  .slider::-moz-range-thumb {
    height: 20px;
    width: 20px;
    border-radius: 50%;
    background: #3B82F6;
    cursor: pointer;
    border: none;
  }
</style>