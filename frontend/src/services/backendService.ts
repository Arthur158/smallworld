import { websocketMessageReceived } from '../redux/slices/applicationSlice';
import { store } from '../redux/store';

let socket: WebSocket | null = null;

// Connect WebSocket
export const connectWebSocket = () => {
  socket = new WebSocket('wss://66e9-188-91-57-106.ngrok-free.app/ws');
   wss://60cd-2a02-2788-11ca-f01-f686-5fff-fe1c-ff6f.ngrok-free.app/ws
  // 'https://397a-2a02-2788-11ca-f01-2e0-4cff-fe68-ed.ngrok-free.app/' 
  // -> 'wss://397a-2a02-2788-11ca-f01-2e0-4cff-fe68-ed.ngrok-free.app/ws'
  // ws://localhost:8080/ws

  socket.onopen = () => {
    console.log('WebSocket connected');
    // Optionally ask for the current rooms list if your server requires a request:
    // sendMessageToBackend('getRooms', {});
  };

  socket.onclose = () => console.log('WebSocket disconnected');
  socket.onerror = (err) => console.error('WebSocket error:', err);

  // Single onmessage handler
  socket.onmessage = (event) => {
    const messageString = event.data; // raw text from server
    console.log("Received from server:", messageString);

    // Dispatch to Redux to handle in applicationSlice (websocketMessageReceived)
    store.dispatch(websocketMessageReceived(messageString));
  };
};

export const sendMessageToBackend = (type: string, data: Record<string, any>) => {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    console.error('WebSocket is not open.');
    return;
  }
  try {
    const msg = {
      type,
      data,
    };
    const payload = JSON.stringify(msg);

    console.log('Sending to server:', payload);
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
