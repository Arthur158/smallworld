import { Polygon } from '../types/Board';
import { websocketMessageReceived } from '../redux/slices/applicationSlice';
import { store } from '../redux/store'; // Import the Redux store

let socket: WebSocket | null = null;

// Connect WebSocket
export const connectWebSocket = (url: string) => {
  socket = new WebSocket(url);

  socket.onopen = () => console.log('WebSocket connected');
  socket.onclose = () => console.log('WebSocket disconnected');
  socket.onerror = (err) => console.error('WebSocket error:', err);

  // Single onmessage handler
  socket.onmessage = (event) => {
    const message = JSON.parse(event.data);
    store.dispatch(websocketMessageReceived(message));
  };
};


export const sendRequest = (message: Record<string, any>) => {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    console.error('WebSocket is not open.');
    return;
  }

  try {
    const payload = JSON.stringify(message); // Convert object to JSON string
    socket.send(payload);
  } catch (err) {
    console.error('Failed to send WebSocket message:', err);
  }
};

// Close WebSocket
export const closeWebSocket = () => {
  if (socket) {
    socket.close();
    socket = null;
    console.log('WebSocket closed');
  } else {
    console.error('WebSocket is not initialized or already closed');
  }
};
