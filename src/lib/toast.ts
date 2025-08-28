/** @file Provides a simple, native toast library. */

import { writable } from "svelte/store";

export const toastStore = writable<(Toast & { expires: number })[]>([]);

export type Toast = {
  kind: "info" | "success" | "error";
  message: string;
  action?: string;
  onAction?: () => void;
  persistent?: boolean; // If true, toast doesn't auto-expire
};

export function makeToast(toast: Toast, duration = 3000) {
  const expires = toast.persistent ? Infinity : Date.now() + duration;
  const obj = Object.assign({ expires }, toast);
  toastStore.update(($toasts) => [...$toasts, obj]);
}

/** Manually dismiss a persistent toast by message content */
export function dismissToast(message: string) {
  toastStore.update(($toasts) => 
    $toasts.filter(toast => toast.message !== message)
  );
}

/** Dismiss all persistent toasts */
export function dismissPersistentToasts() {
  toastStore.update(($toasts) => 
    $toasts.filter(toast => !toast.persistent)
  );
}
