import { io, Socket } from "socket.io-client";
import { onUnmounted, ref, getCurrentInstance } from "vue";

const socket = ref<Socket | null>(null);

export function useSocket() {
  if (!socket.value) {
    socket.value = io("http://localhost:8080", {
      transports: ["websocket"],
      withCredentials: true,
    });
  }

  /**
   * Listens to an event and returns an unsubscribe function.
   * Auto-cleanup only happens if called inside setup().
   */
  function onEvent(event: string, callback: (data: any) => void) {
    if (!socket.value) return () => { };

    const wrapper = (data: any) => {
      callback(data);
    };

    socket.value.on(event, wrapper);

    const off = () => {
      socket.value?.off(event, wrapper);
    };

    // Only attach auto-cleanup if we are inside a component setup context
    if (getCurrentInstance()) {
      onUnmounted(off);
    }

    return off;
  }

  return { socket, onEvent };
}
