import { browser } from "$app/environment";
import { get } from "svelte/store";
import { settings, updateSettings, type Settings } from "./settings";
import { wallpaperManager, wallpaperSettings, type Wallpaper, type WallpaperSettings } from "./wallpaper";

const PROFILE_EXPORT_VERSION = "1.0.0";

export interface ProfileExport {
  version: string;
  exportedAt: string;
  exportedBy: string;
  profile: {
    settings: Settings;
    customWallpapers: Wallpaper[];
    wallpaperSettings: WallpaperSettings;
  };
}

export interface ImportOptions {
  includeSettings: boolean;
  includeWallpapers: boolean;
  overwriteExistingWallpapers: boolean;
}

export interface ImportValidationResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
  preview: {
    settingsCount: number;
    wallpapersCount: number;
    exportedAt: string;
    exportedBy: string;
    version: string;
  };
}

export class ProfileManager {
  /**
   * Export all user profile data to a JSON file
   */
  async exportProfile(): Promise<void> {
    if (!browser) {
      throw new Error("Export is only available in the browser");
    }

    try {
      const currentSettings = get(settings);
      const currentWallpaperSettings = get(wallpaperSettings);
      const customWallpapers = wallpaperManager.getCustomWallpapers();

      const exportData: ProfileExport = {
        version: PROFILE_EXPORT_VERSION,
        exportedAt: new Date().toISOString(),
        exportedBy: currentSettings.name || "Unknown User",
        profile: {
          settings: currentSettings,
          customWallpapers,
          wallpaperSettings: currentWallpaperSettings,
        },
      };

      const blob = new Blob([JSON.stringify(exportData, null, 2)], {
        type: "application/json",
      });

      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `sshx-profile-${currentSettings.name || "user"}-${
        new Date().toISOString().split("T")[0]
      }.json`;
      
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error("Failed to export profile:", error);
      throw new Error("Failed to export profile data");
    }
  }

  /**
   * Validate an imported profile file
   */
  validateImportFile(fileContent: string): ImportValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];

    try {
      const data: ProfileExport = JSON.parse(fileContent);

      // Check required fields
      if (!data.version) {
        errors.push("Missing version field");
      } else if (!this.isVersionCompatible(data.version)) {
        errors.push(`Incompatible version: ${data.version}. Expected: ${PROFILE_EXPORT_VERSION}`);
      }

      if (!data.exportedAt) {
        warnings.push("Missing export timestamp");
      }

      if (!data.profile) {
        errors.push("Missing profile data");
        return {
          valid: false,
          errors,
          warnings,
          preview: {
            settingsCount: 0,
            wallpapersCount: 0,
            exportedAt: data.exportedAt || "Unknown",
            exportedBy: data.exportedBy || "Unknown",
            version: data.version || "Unknown",
          },
        };
      }

      // Validate settings
      if (data.profile.settings) {
        const settingsValidation = this.validateSettings(data.profile.settings);
        if (settingsValidation.length > 0) {
          warnings.push(...settingsValidation);
        }
      }

      // Validate wallpapers
      const wallpapersCount = data.profile.customWallpapers?.length || 0;
      if (wallpapersCount > 0) {
        const wallpaperValidation = this.validateWallpapers(data.profile.customWallpapers);
        if (wallpaperValidation.length > 0) {
          warnings.push(...wallpaperValidation);
        }

        // Check storage implications
        const totalSize = this.estimateWallpaperStorageSize(data.profile.customWallpapers);
        if (totalSize > 5 * 1024 * 1024) { // 5MB
          warnings.push(`Custom wallpapers are large (${this.formatFileSize(totalSize)}). May exceed storage limits.`);
        }
      }

      const settingsCount = data.profile.settings ? Object.keys(data.profile.settings).length : 0;

      return {
        valid: errors.length === 0,
        errors,
        warnings,
        preview: {
          settingsCount,
          wallpapersCount,
          exportedAt: data.exportedAt || "Unknown",
          exportedBy: data.exportedBy || "Unknown",
          version: data.version || "Unknown",
        },
      };
    } catch (error) {
      return {
        valid: false,
        errors: ["Invalid JSON file format"],
        warnings: [],
        preview: {
          settingsCount: 0,
          wallpapersCount: 0,
          exportedAt: "Unknown",
          exportedBy: "Unknown",
          version: "Unknown",
        },
      };
    }
  }

  /**
   * Import profile data with specified options
   */
  async importProfile(fileContent: string, options: ImportOptions): Promise<void> {
    if (!browser) {
      throw new Error("Import is only available in the browser");
    }

    const validation = this.validateImportFile(fileContent);
    if (!validation.valid) {
      throw new Error(`Import validation failed: ${validation.errors.join(", ")}`);
    }

    try {
      const data: ProfileExport = JSON.parse(fileContent);

      // Import settings
      if (options.includeSettings && data.profile.settings) {
        const sanitizedSettings = this.sanitizeSettings(data.profile.settings);
        updateSettings(sanitizedSettings);
      }

      // Import wallpapers
      if (options.includeWallpapers && data.profile.customWallpapers) {
        await this.importWallpapers(data.profile.customWallpapers, options.overwriteExistingWallpapers);
        
        // Import wallpaper settings
        if (data.profile.wallpaperSettings) {
          wallpaperSettings.update(() => data.profile.wallpaperSettings);
        }
      }
    } catch (error) {
      console.error("Failed to import profile:", error);
      throw new Error("Failed to import profile data");
    }
  }

  /**
   * Check if the export version is compatible
   */
  private isVersionCompatible(version: string): boolean {
    const [major] = version.split(".").map(Number);
    const [currentMajor] = PROFILE_EXPORT_VERSION.split(".").map(Number);
    return major === currentMajor;
  }

  /**
   * Validate settings object
   */
  private validateSettings(settings: any): string[] {
    const warnings: string[] = [];

    // Check for potentially unsafe settings
    if (settings.geminiApiKey && typeof settings.geminiApiKey === "string" && settings.geminiApiKey.length > 100) {
      warnings.push("Gemini API key appears to be included. Verify this is intentional.");
    }

    if (settings.openRouterApiKey && typeof settings.openRouterApiKey === "string" && settings.openRouterApiKey.length > 100) {
      warnings.push("OpenRouter API key appears to be included. Verify this is intentional.");
    }

    // Check for reasonable value ranges
    if (settings.fontSize && (settings.fontSize < 8 || settings.fontSize > 32)) {
      warnings.push("Font size is outside normal range (8-32px)");
    }

    if (settings.scrollback && settings.scrollback < 0) {
      warnings.push("Scrollback value is negative");
    }

    if (settings.wallpaperOpacity && (settings.wallpaperOpacity < 0.1 || settings.wallpaperOpacity > 1.0)) {
      warnings.push("Wallpaper opacity is outside valid range (0.1-1.0)");
    }

    return warnings;
  }

  /**
   * Validate wallpapers array
   */
  private validateWallpapers(wallpapers: any[]): string[] {
    const warnings: string[] = [];

    wallpapers.forEach((wallpaper, index) => {
      if (!wallpaper.id) {
        warnings.push(`Wallpaper ${index + 1} is missing an ID`);
      }
      if (!wallpaper.url || !wallpaper.url.startsWith("data:image/")) {
        warnings.push(`Wallpaper ${index + 1} has invalid image data`);
      }
      if (!wallpaper.name) {
        warnings.push(`Wallpaper ${index + 1} is missing a name`);
      }
    });

    return warnings;
  }

  /**
   * Estimate total storage size of wallpapers
   */
  private estimateWallpaperStorageSize(wallpapers: Wallpaper[]): number {
    return wallpapers.reduce((total, wallpaper) => {
      if (wallpaper.url && wallpaper.url.startsWith("data:")) {
        // Rough estimate: base64 is ~33% larger than binary
        const base64Data = wallpaper.url.split(",")[1] || "";
        return total + (base64Data.length * 0.75);
      }
      return total;
    }, 0);
  }

  /**
   * Format file size for display
   */
  private formatFileSize(bytes: number): string {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  }

  /**
   * Sanitize imported settings to ensure they're safe
   */
  private sanitizeSettings(importedSettings: any): Partial<Settings> {
    const current = get(settings);
    const sanitized: any = {};

    // Copy safe settings with validation
    const safeFields: (keyof Settings)[] = [
      "name", "theme", "uiTheme", "scrollback", "fontFamily", "fontSize", "fontWeight", "fontWeightBold",
      "toolbarPosition", "zoomLevel", "copyOnSelect", "middleClickPaste", "copyButtonEnabled", "copyButtonFormat",
      "downloadButtonEnabled", "downloadButtonBehavior", "screenshotButtonEnabled", "aiEnabled", "aiProvider",
      "aiModel", "aiModels", "openRouterModel", "openRouterModels", "aiContextLength", "aiAutoCompress", "aiMaxResponseTokens",
      "wallpaperEnabled", "wallpaperCurrent", "wallpaperFit", "wallpaperOpacity", "titlebarSeparator", "titlebarSeparatorColor",
      "titlebarColor", "titlebarColorEnabled", "terminalsBarEnabled", "terminalsBarPosition", "idleDisconnectEnabled"
    ];

    safeFields.forEach(field => {
      if (importedSettings[field] !== undefined) {
        sanitized[field] = importedSettings[field];
      }
    });

    // Handle API keys with user confirmation (don't auto-import for security)
    // Users would need to manually re-enter API keys after import

    // Validate ranges and types
    if (sanitized.fontSize !== undefined) {
      sanitized.fontSize = Math.max(8, Math.min(32, Number(sanitized.fontSize) || current.fontSize));
    }

    if (sanitized.scrollback !== undefined) {
      sanitized.scrollback = Math.max(0, Number(sanitized.scrollback) || current.scrollback);
    }

    if (sanitized.wallpaperOpacity !== undefined) {
      sanitized.wallpaperOpacity = Math.max(0.1, Math.min(1.0, Number(sanitized.wallpaperOpacity) || current.wallpaperOpacity));
    }

    return sanitized;
  }

  /**
   * Import custom wallpapers
   */
  private async importWallpapers(wallpapers: Wallpaper[], overwriteExisting: boolean): Promise<void> {
    if (!wallpapers || wallpapers.length === 0) {
      return;
    }

    const existingWallpapers = wallpaperManager.getCustomWallpapers();
    const existingIds = new Set(existingWallpapers.map(w => w.id));

    for (const wallpaper of wallpapers) {
      // Skip if wallpaper exists and we're not overwriting
      if (existingIds.has(wallpaper.id) && !overwriteExisting) {
        continue;
      }

      // Remove existing wallpaper if overwriting
      if (existingIds.has(wallpaper.id) && overwriteExisting) {
        wallpaperManager.removeCustomWallpaper(wallpaper.id);
      }

      // Create a new file from the base64 data and add it
      try {
        if (wallpaper.url && wallpaper.url.startsWith('data:image/')) {
          const response = await fetch(wallpaper.url);
          const blob = await response.blob();
          const file = new File([blob], wallpaper.name || 'imported-wallpaper', { type: blob.type });
          
          const newWallpaper = await wallpaperManager.addCustomWallpaper(file);
          console.log(`Successfully imported wallpaper: ${wallpaper.name}`);
        }
      } catch (error) {
        console.warn(`Failed to import wallpaper "${wallpaper.name}":`, error);
      }
    }
  }
}

export const profileManager = new ProfileManager();