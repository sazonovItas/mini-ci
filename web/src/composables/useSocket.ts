import { io, Socket } from "socket.io-client";
import { onUnmounted, ref, getCurrentInstance } from "vue";

const socket = ref<Socket | null>(null);

export function useSocket() {
  if (!socket.value) {
    const socketUrl = import.meta.env.VITE_SOCKET_URL || 'http://localhost:8080';

    socket.value = io(socketUrl, {
      transports: ["websocket"],
      withCredentials: true,
    });

    socket.value.on("connect", () => console.log("Socket Connected"));
  }

  function onEvent(event: string, callback: (data: any) => void) {
    if (!socket.value) return () => { };

    const wrapper = (data: any) => {
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
