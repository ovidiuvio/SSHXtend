import { browser } from "$app/environment";
import { persisted } from "svelte-persisted-store";

export type WallpaperType = "none" | "builtin" | "custom";
export type WallpaperFit = "cover" | "contain" | "fill" | "tile" | "center";

export interface Wallpaper {
  id: string;
  name: string;
  type: WallpaperType;
  url: string;
  thumbnail?: string;
  builtIn?: boolean;
}

export interface WallpaperSettings {
  currentWallpaper: string; // wallpaper id
  fit: WallpaperFit;
  opacity: number; // 0.1 to 1.0
}

// Built-in wallpapers with popular color palette variations
export const BUILTIN_WALLPAPERS: Wallpaper[] = [
  {
    id: "none",
    name: "None (Dots Pattern)",
    type: "none",
    url: "",
    builtIn: true,
  },
  
  // Ubuntu Base Collection
  {
    id: "ubuntu-default",
    name: "Ubuntu Default",
    type: "builtin",
    url: "/wallpapers/ubuntu-default.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-default.svg",
    builtIn: true,
  },
  {
    id: "ubuntu-purple",
    name: "Ubuntu Purple",
    type: "builtin", 
    url: "/wallpapers/ubuntu-purple.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-purple.svg",
    builtIn: true,
  },
  {
    id: "ubuntu-dark",
    name: "Ubuntu Dark",
    type: "builtin",
    url: "/wallpapers/ubuntu-dark.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-dark.svg",
    builtIn: true,
  },
  {
    id: "ubuntu-sunrise",
    name: "Ubuntu Sunrise",
    type: "builtin",
    url: "/wallpapers/ubuntu-sunrise.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-sunrise.svg",
    builtIn: true,
  },
  {
    id: "ubuntu-cosmic",
    name: "Ubuntu Cosmic",
    type: "builtin",
    url: "/wallpapers/ubuntu-cosmic.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-cosmic.svg",
    builtIn: true,
  },
  {
    id: "ubuntu-minimal",
    name: "Ubuntu Minimal",
    type: "builtin",
    url: "/wallpapers/ubuntu-minimal.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-minimal.svg",
    builtIn: true,
  },
  
  // Ubuntu Default Style Variations
  {
    id: "ubuntu-default-dracula",
    name: "Ubuntu Dracula",
    type: "builtin",
    url: "/wallpapers/ubuntu-default-dracula.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-default-dracula.svg",
    builtIn: true,
  },
  {
    id: "ubuntu-default-nord",
    name: "Ubuntu Nord",
    type: "builtin",
    url: "/wallpapers/ubuntu-default-nord.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-default-nord.svg",
    builtIn: true,
  },
  {
    id: "ubuntu-default-solarized-light",
    name: "Ubuntu Solarized Light",
    type: "builtin",
    url: "/wallpapers/ubuntu-default-solarized-light.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-default-solarized-light.svg",
    builtIn: true,
  },
  {
    id: "ubuntu-default-tokyo-night",
    name: "Ubuntu Tokyo Night",
    type: "builtin",
    url: "/wallpapers/ubuntu-default-tokyo-night.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-default-tokyo-night.svg",
    builtIn: true,
  },
  {
    id: "ubuntu-default-gruvbox",
    name: "Ubuntu Gruvbox",
    type: "builtin",
    url: "/wallpapers/ubuntu-default-gruvbox.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-default-gruvbox.svg",
    builtIn: true,
  },
  
  // Ubuntu Dark Style Variations
  {
    id: "ubuntu-dark-catppuccin",
    name: "Ubuntu Catppuccin",
    type: "builtin",
    url: "/wallpapers/ubuntu-dark-catppuccin.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-dark-catppuccin.svg",
    builtIn: true,
  },
  {
    id: "ubuntu-dark-onedark",
    name: "Ubuntu One Dark",
    type: "builtin",
    url: "/wallpapers/ubuntu-dark-onedark.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-dark-onedark.svg",
    builtIn: true,
  },
  {
    id: "ubuntu-dark-monokai",
    name: "Ubuntu Monokai",
    type: "builtin",
    url: "/wallpapers/ubuntu-dark-monokai.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-dark-monokai.svg",
    builtIn: true,
  },
  {
    id: "ubuntu-dark-material",
    name: "Ubuntu Material Dark",
    type: "builtin",
    url: "/wallpapers/ubuntu-dark-material.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-dark-material.svg",
    builtIn: true,
  },
  {
    id: "ubuntu-dark-palenight",
    name: "Ubuntu Palenight",
    type: "builtin",
    url: "/wallpapers/ubuntu-dark-palenight.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-dark-palenight.svg",
    builtIn: true,
  },
  
  // Ubuntu Minimal Style Variations
  {
    id: "ubuntu-minimal-everforest",
    name: "Ubuntu Everforest",
    type: "builtin",
    url: "/wallpapers/ubuntu-minimal-everforest.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-minimal-everforest.svg",
    builtIn: true,
  },
  {
    id: "ubuntu-minimal-rosepine",
    name: "Ubuntu Rosé Pine",
    type: "builtin",
    url: "/wallpapers/ubuntu-minimal-rosepine.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-minimal-rosepine.svg",
    builtIn: true,
  },
  {
    id: "ubuntu-minimal-github-dark",
    name: "Ubuntu GitHub Dark",
    type: "builtin",
    url: "/wallpapers/ubuntu-minimal-github-dark.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-minimal-github-dark.svg",
    builtIn: true,
  },
  {
    id: "ubuntu-minimal-horizon",
    name: "Ubuntu Horizon",
    type: "builtin",
    url: "/wallpapers/ubuntu-minimal-horizon.svg",
    thumbnail: "/wallpapers/thumbs/ubuntu-minimal-horizon.svg",
    builtIn: true,
  },
];

const WALLPAPER_STORAGE_KEY = "sshx-wallpapers";
const WALLPAPER_SETTINGS_KEY = "sshx-wallpaper-settings";

// Persisted store for wallpaper settings
export const wallpaperSettings = persisted<WallpaperSettings>(WALLPAPER_SETTINGS_KEY, {
  currentWallpaper: "none",
  fit: "cover",
  opacity: 1.0,
});

export class WallpaperManager {
  private customWallpapers: Wallpaper[] = [];

  constructor() {
    if (browser) {
      this.loadCustomWallpapers();
    }
  }

  /**
   * Get all available wallpapers (builtin + custom)
   */
  getAllWallpapers(): Wallpaper[] {
    return [...BUILTIN_WALLPAPERS, ...this.customWallpapers];
  }

  /**
   * Get builtin wallpapers only
   */
  getBuiltinWallpapers(): Wallpaper[] {
    return BUILTIN_WALLPAPERS;
  }

  /**
   * Get custom wallpapers only
   */
  getCustomWallpapers(): Wallpaper[] {
    return this.customWallpapers;
  }

  /**
   * Get a wallpaper by ID
   */
  getWallpaper(id: string): Wallpaper | null {
    const all = this.getAllWallpapers();
    return all.find(w => w.id === id) || null;
  }

  /**
   * Add a custom wallpaper from file
   */
  async addCustomWallpaper(file: File): Promise<Wallpaper | null> {
    if (!browser) return null;

    // Validate file type
    if (!file.type.startsWith('image/')) {
      throw new Error('File must be an image');
    }

    // Validate file size (max 5MB)
    if (file.size > 5 * 1024 * 1024) {
      throw new Error('Image must be smaller than 5MB');
    }

    try {
      // Convert to base64 data URL
      const dataUrl = await this.fileToDataUrl(file);
      
      // Generate thumbnail
      const thumbnail = await this.generateThumbnail(dataUrl, 200, 120);
      
      // Create wallpaper object
      const wallpaper: Wallpaper = {
        id: `custom-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
        name: file.name.replace(/\.[^/.]+$/, ""), // Remove extension
        type: "custom",
        url: dataUrl,
        thumbnail,
        builtIn: false,
      };

      // Add to collection
      this.customWallpapers.push(wallpaper);
      this.saveCustomWallpapers();

      return wallpaper;
    } catch (error) {
      console.error('Failed to add custom wallpaper:', error);
      throw error;
    }
  }

  /**
   * Remove a custom wallpaper
   */
  removeCustomWallpaper(id: string): boolean {
    const index = this.customWallpapers.findIndex(w => w.id === id);
    if (index === -1) return false;

    this.customWallpapers.splice(index, 1);
    this.saveCustomWallpapers();
    return true;
  }

  /**
   * Generate CSS background style for a wallpaper
   */
  generateBackgroundStyle(
    wallpaper: Wallpaper | null,
    fit: WallpaperFit,
    opacity: number,
    zoom: number,
    center: readonly number[] | [number, number]
  ): string {
    if (!wallpaper || wallpaper.type === "none") {
      // Return the original dotted pattern
      const centerX = center[0] ?? 0;
      const centerY = center[1] ?? 0;
      return `
        background-image: radial-gradient(rgb(var(--color-border)) ${zoom}px, transparent 0);
        background-size: ${24 * zoom}px ${24 * zoom}px;
        background-position: ${-zoom * centerX}px ${-zoom * centerY}px;
        background-color: rgb(var(--color-bg));
      `.trim();
    }

    const backgroundSize = this.getFitStyle(fit);
    const centerX = center[0] ?? 0;
    const centerY = center[1] ?? 0;
    const backgroundPosition = fit === "tile" ? `${-zoom * centerX}px ${-zoom * centerY}px` : "center center";
    
    return `
      background-image: url('${wallpaper.url}');
      background-size: ${backgroundSize};
      background-position: ${backgroundPosition};
      background-repeat: ${fit === "tile" ? "repeat" : "no-repeat"};
      background-attachment: fixed;
      opacity: ${opacity};
      background-color: rgb(var(--color-bg));
    `.trim();
  }

  /**
   * Generate thumbnail from image data URL
   */
  private async generateThumbnail(imageUrl: string, width: number, height: number): Promise<string> {
    return new Promise((resolve, reject) => {
      const img = new Image();
      img.onload = () => {
        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');
        if (!ctx) {
          reject(new Error('Failed to get canvas context'));
          return;
        }

        canvas.width = width;
        canvas.height = height;

        // Calculate dimensions to maintain aspect ratio
        const imgAspect = img.width / img.height;
        const canvasAspect = width / height;

        let drawWidth = width;
        let drawHeight = height;
        let drawX = 0;
        let drawY = 0;

        if (imgAspect > canvasAspect) {
          // Image is wider than canvas
          drawHeight = height;
          drawWidth = height * imgAspect;
          drawX = (width - drawWidth) / 2;
        } else {
          // Image is taller than canvas
          drawWidth = width;
          drawHeight = width / imgAspect;
          drawY = (height - drawHeight) / 2;
        }

        // Fill with dark background
        ctx.fillStyle = '#1f1f1f';
        ctx.fillRect(0, 0, width, height);

        // Draw scaled image
        ctx.drawImage(img, drawX, drawY, drawWidth, drawHeight);

        resolve(canvas.toDataURL('image/jpeg', 0.8));
      };
      img.onerror = () => reject(new Error('Failed to load image for thumbnail'));
      img.src = imageUrl;
    });
  }

  /**
   * Convert file to data URL
   */
  private fileToDataUrl(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(reader.result as string);
      reader.onerror = () => reject(new Error('Failed to read file'));
      reader.readAsDataURL(file);
    });
  }

  /**
   * Get CSS background-size value for fit style
   */
  private getFitStyle(fit: WallpaperFit): string {
    switch (fit) {
      case "cover":
        return "cover";
      case "contain":
        return "contain";
      case "fill":
        return "100% 100%";
      case "tile":
        return "auto";
      case "center":
        return "auto";
      default:
        return "cover";
    }
  }

  /**
   * Load custom wallpapers from localStorage
   */
  private loadCustomWallpapers(): void {
    try {
      const stored = localStorage.getItem(WALLPAPER_STORAGE_KEY);
      if (stored) {
        this.customWallpapers = JSON.parse(stored);
      }
    } catch (error) {
      console.error('Failed to load custom wallpapers:', error);
      this.customWallpapers = [];
    }
  }

  /**
   * Save custom wallpapers to localStorage
   */
  private saveCustomWallpapers(): void {
    try {
      localStorage.setItem(WALLPAPER_STORAGE_KEY, JSON.stringify(this.customWallpapers));
    } catch (error) {
      console.error('Failed to save custom wallpapers:', error);
    }
  }

  /**
   * Get storage usage for custom wallpapers
   */
  getStorageUsage(): { used: number; total: number; percentage: number } {
    if (!browser) return { used: 0, total: 0, percentage: 0 };

    try {
      const stored = localStorage.getItem(WALLPAPER_STORAGE_KEY) || '[]';
      const used = new Blob([stored]).size;
      const total = 5 * 1024 * 1024; // Assume 5MB localStorage limit
      const percentage = Math.round((used / total) * 100);

      return { used, total, percentage };
    } catch (error) {
      return { used: 0, total: 0, percentage: 0 };
    }
  }

  /**
   * Clear all custom wallpapers
   */
  clearCustomWallpapers(): void {
    this.customWallpapers = [];
    if (browser) {
      localStorage.removeItem(WALLPAPER_STORAGE_KEY);
    }
  }
}

// Global wallpaper manager instance
export const wallpaperManager = new WallpaperManager();

// Utility function to update wallpaper settings
export function updateWallpaperSettings(updates: Partial<WallpaperSettings>) {
  wallpaperSettings.update(settings => ({ ...settings, ...updates }));
}