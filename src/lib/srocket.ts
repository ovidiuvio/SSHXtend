/**
 * @file Internal library for sshx, providing real-time communication.
 *
 * The contents of this file are technically general, not sshx-specific, but it
 * is not open-sourced as its own library because it's not ready for that.
 */

import { encode, decode } from "cbor-x";

/** How long to wait between reconnections (in milliseconds). */
const RECONNECT_DELAY = 500;

/** Number of messages to queue while disconnected. */
const BUFFER_SIZE = 64;

export type SrocketOptions<T> = {
  /** Handle a message received from the server. */
  onMessage(message: T): void;

  /** Called when the socket connects to the server. */
  onConnect?(): void;

  /** Called when a connected socket is closed. */
  onDisconnect?(): void;

  /** Called when an incoming or existing connection is closed. */
  onClose?(event: CloseEvent): void;
};

/** A reconnecting WebSocket client for real-time communication. */
export class Srocket<T, U> {
  #url: string;
  #options: SrocketOptions<T>;

  #ws: WebSocket | null;
  #connected: boolean;
  #buffer: Uint8Array[];
  #disposed: boolean;
  #idleDisconnected: boolean;

  constructor(url: string, options: SrocketOptions<T>) {
    this.#url = url;
    if (this.#url.startsWith("/")) {
      // Get WebSocket URL relative to the current origin.
      this.#url =
        (window.location.protocol === "https:" ? "wss://" : "ws://") +
        window.location.host +
        this.#url;
    }
    this.#options = options;

    this.#ws = null;
    this.#connected = false;
    this.#buffer = [];
    this.#disposed = false;
    this.#idleDisconnected = false;
    this.#reconnect();
  }

  get connected() {
    return this.#connected;
  }

  get idleDisconnected() {
    return this.#idleDisconnected;
  }

  /** Queue a message to send to the server, with "at-most-once" semantics. */
  send(message: U) {
    // Types in cbor-x are incorrect here, so cast to fix the error.
    // See: https://github.com/kriszyp/cbor-x/issues/120
    const data = <Uint8Array>(encode(message) as unknown);

    if (this.#connected && this.#ws) {
      this.#ws.send(data);
    } else {
      if (this.#buffer.length < BUFFER_SIZE) {
        this.#buffer.push(data);
      }
    }
  }

  /** Disconnect the WebSocket for idle detection. */
  disconnect() {
    if (this.#disposed) return;
    this.#idleDisconnected = true;
    this.#stateChange(false);
    this.#ws?.close();
  }

  /** Reconnect the WebSocket when resuming from idle. */
  reconnect() {
    if (this.#disposed) return;
    if (this.#idleDisconnected) {
      this.#idleDisconnected = false;
      // Only reconnect if not already connected/connecting
      if (!this.#ws) {
        this.#reconnect();
      }
    }
  }

  /** Dispose of this WebSocket permanently. */
  dispose() {
    this.#stateChange(false);
    this.#disposed = true;
    this.#ws?.close();
  }

  #reconnect() {
    if (this.#disposed || this.#idleDisconnected) return;
    if (this.#ws !== null) {
      throw new Error("invariant violation: reconnecting while connected");
    }
    this.#ws = new WebSocket(this.#url);
    this.#ws.binaryType = "arraybuffer";
    this.#ws.onopen = () => {
      this.#stateChange(true);
    };
    this.#ws.onclose = (event) => {
      this.#options.onClose?.(event);
      this.#ws = null;
      this.#stateChange(false);
      // Only auto-reconnect if not idle disconnected
      if (!this.#idleDisconnected) {
        setTimeout(() => this.#reconnect(), RECONNECT_DELAY);
      }
    };
    this.#ws.onmessage = (event) => {
      if (event.data instanceof ArrayBuffer) {
        const message: T = decode(new Uint8Array(event.data));
        this.#options.onMessage(message);
      } else {
        console.warn("unexpected non-buffer message, ignoring");
      }
    };
  }

  #stateChange(connected: boolean) {
    if (!this.#disposed && connected !== this.#connected) {
      this.#connected = connected;
      if (connected) {
        this.#options.onConnect?.();

        if (!this.#ws) {
          throw new Error("invariant violation: connected but ws is null");
        }
        // Send any queued messages.
        for (const message of this.#buffer) {
          this.#ws.send(message);
        }
        this.#buffer = [];
      } else {
        this.#options.onDisconnect?.();
      }
    }
  }
}
