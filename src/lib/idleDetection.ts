/**
 * @file Idle detection system for automatic WebSocket disconnect/reconnect
 * 
 * Monitors user activity (mouse, keyboard, touch) and automatically
 * disconnects WebSocket connections when idle to save bandwidth.
 * Reconnects automatically when activity resumes.
 */

import { writable, type Readable } from "svelte/store";
import { browser } from "$app/environment";

/** Default idle timeout for single user (in milliseconds). */
const SINGLE_USER_TIMEOUT = 3 * 60 * 1000; // 3 minutes

/** Extended idle timeout for multiple users (in milliseconds). */
const MULTI_USER_TIMEOUT = 10 * 60 * 1000; // 10 minutes

/** Tracks if the session is currently idle */
export const isIdle = writable<boolean>(false);

/** Tracks if idle disconnect is currently active (disconnected due to idle) */
export const isIdleDisconnected = writable<boolean>(false);

export class IdleDetectionManager {
  #enabled: boolean;
  #lastActivity: number;
  #idleTimer: number | null = null;
  #onIdle: () => void;
  #onActive: () => void;
  #isCurrentlyIdle: boolean = false;
  #eventListeners: Array<{ element: EventTarget; event: string; handler: EventListener }> = [];
  #currentTimeout: number = SINGLE_USER_TIMEOUT;

  constructor(onIdle: () => void, onActive: () => void, enabled: boolean = true) {
    this.#onIdle = onIdle;
    this.#onActive = onActive;
    this.#enabled = enabled;
    this.#lastActivity = Date.now();

    if (browser && enabled) {
      this.#setupEventListeners();
      this.#startIdleTimer();
    }
  }

  /**
   * Enable or disable idle detection
   */
  setEnabled(enabled: boolean) {
    if (this.#enabled === enabled) return;
    
    this.#enabled = enabled;
    
    if (enabled && browser) {
      this.#setupEventListeners();
      this.#startIdleTimer();
      // Reset to active state when enabling
      if (this.#isCurrentlyIdle) {
        this.#handleActivity();
      }
    } else {
      this.#cleanup();
      // If disabling and currently idle, mark as active
      if (this.#isCurrentlyIdle) {
        this.#isCurrentlyIdle = false;
        isIdle.set(false);
        isIdleDisconnected.set(false);
        this.#onActive();
      }
    }
  }

  /**
   * Manually trigger activity (useful for WebSocket message events)
   */
  recordActivity() {
    if (!this.#enabled) return;
    this.#handleActivity();
  }

  /**
   * Update the idle timeout based on user count
   */
  setIdleTimeout(userCount: number) {
    const newTimeout = userCount > 1 ? MULTI_USER_TIMEOUT : SINGLE_USER_TIMEOUT;
    
    if (newTimeout !== this.#currentTimeout) {
      this.#currentTimeout = newTimeout;
      
      // If currently active, restart the timer with new timeout
      if (!this.#isCurrentlyIdle && this.#enabled) {
        this.#startIdleTimer();
      }
    }
  }

  /**
   * Get current idle timeout in milliseconds
   */
  get currentTimeout() {
    return this.#currentTimeout;
  }

  /**
   * Get current idle state
   */
  get idle() {
    return this.#isCurrentlyIdle;
  }

  /**
   * Setup event listeners for user activity detection
   */
  #setupEventListeners() {
    if (!browser) return;

    const events = [
      'mousedown',
      'mousemove', 
      'mouseup',
      'click',
      'keydown',
      'keyup',
      'keypress',
      'scroll',
      'touchstart',
      'touchmove',
      'touchend',
      'wheel',
      'focus',
      'blur'
    ];

    const handler = (event: Event) => {
      // Throttle mousemove events to avoid excessive calls
      if (event.type === 'mousemove') {
        if (Date.now() - this.#lastActivity < 1000) return; // Only update every second for mousemove
      }
      this.#handleActivity();
    };

    events.forEach(eventName => {
      // Listen on document for broader coverage
      document.addEventListener(eventName, handler, { passive: true });
      this.#eventListeners.push({ element: document, event: eventName, handler });
    });

    // Also listen on window for visibility changes
    const visibilityHandler = () => {
      if (!document.hidden) {
        this.#handleActivity();
      }
    };
    
    document.addEventListener('visibilitychange', visibilityHandler);
    this.#eventListeners.push({ element: document, event: 'visibilitychange', handler: visibilityHandler });
  }

  /**
   * Handle user activity - reset idle timer and mark as active
   */
  #handleActivity() {
    if (!this.#enabled) return;
    
    const now = Date.now();
    this.#lastActivity = now;

    // If we were idle, mark as active and trigger callback
    if (this.#isCurrentlyIdle) {
      this.#isCurrentlyIdle = false;
      isIdle.set(false);
      isIdleDisconnected.set(false);
      this.#onActive();
    }

    // Reset the idle timer
    this.#startIdleTimer();
  }

  /**
   * Start the idle detection timer
   */
  #startIdleTimer() {
    if (!this.#enabled) return;

    // Clear existing timer
    if (this.#idleTimer) {
      clearTimeout(this.#idleTimer);
    }

    // Set new timer
    this.#idleTimer = window.setTimeout(() => {
      if (!this.#enabled) return;
      
      const now = Date.now();
      const timeSinceLastActivity = now - this.#lastActivity;

      // Double-check we're actually idle (in case of race conditions)
      if (timeSinceLastActivity >= this.#currentTimeout && !this.#isCurrentlyIdle) {
        this.#isCurrentlyIdle = true;
        isIdle.set(true);
        isIdleDisconnected.set(true);
        this.#onIdle();
      }
    }, this.#currentTimeout);
  }

  /**
   * Clean up event listeners and timers
   */
  #cleanup() {
    if (this.#idleTimer) {
      clearTimeout(this.#idleTimer);
      this.#idleTimer = null;
    }

    this.#eventListeners.forEach(({ element, event, handler }) => {
      element.removeEventListener(event, handler);
    });
    this.#eventListeners = [];
  }

  /**
   * Dispose of the idle detection manager
   */
  dispose() {
    this.#cleanup();
    this.#enabled = false;
    isIdle.set(false);
    isIdleDisconnected.set(false);
  }
}

// Export a singleton instance that can be used across the app
let globalIdleManager: IdleDetectionManager | null = null;

export function createIdleManager(onIdle: () => void, onActive: () => void, enabled: boolean = true): IdleDetectionManager {
  // Clean up existing manager if any
  if (globalIdleManager) {
    globalIdleManager.dispose();
  }
  
  globalIdleManager = new IdleDetectionManager(onIdle, onActive, enabled);
  return globalIdleManager;
}

export function getIdleManager(): IdleDetectionManager | null {
  return globalIdleManager;
}