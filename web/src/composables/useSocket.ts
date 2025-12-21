import { io, Socket } from "socket.io-client";
import { onUnmounted, ref, getCurrentInstance } from "vue";

const socket = ref<Socket | null>(null);

export function useSocket() {
  if (!socket.value) {
    socket.value = io("http://localhost:8080", {
      transports: ["websocket"],
      withCredentials: true,
    });

    socket.value.on("connect", () => console.log("Socket Connected"));
  }

  function onEvent(event: string, callback: (data: any) => void) {
    if (!socket.value) return () => { };

    const wrapper = (data: any) => {
      // Debug log to verify we are receiving events
      console.log(`📩 Event [${event}] received:`, data);

      if (data && data.payload) {
        callback(data.payload);
      } else {
        callback(data);
      }
    };

    socket.value.on(event, wrapper);

    const off = () => {
      console.log(`🔌 Unsubscribing from [${event}]`);
      socket.value?.off(event, wrapper);
    };

    if (getCurrentInstance()) {
      onUnmounted(off);
    }

    return off;
  }

  return { socket, onEvent };
}
