import { websocketMessageReceived, mapEditorFailed } from '../redux/slices/applicationSlice';
import { store } from '../redux/store';

let socket: WebSocket | null = null;

export interface AuthConfig {
  enabled: boolean;
  providerName?: string;
}

export const getAuthConfig = async (): Promise<AuthConfig> => {
  const response = await fetch('/auth/config', {
    credentials: 'same-origin',
    headers: { Accept: 'application/json' },
  });
  if (!response.ok) {
    throw new Error(`Unable to load authentication configuration (${response.status})`);
  }
  return response.json();
};

// Connect WebSocket
export const connectWebSocket = () => {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  socket = new WebSocket(`${protocol}//${window.location.host}/ws`);

  socket.onopen = () => {
    console.log('WebSocket connected');
    // Optionally ask for the current rooms list if your server requires a request:
    // sendMessageToBackend('getRooms', {});
  };

  socket.onclose = () => {
    console.log('WebSocket disconnected');
    store.dispatch(mapEditorFailed('Connection lost. Reconnect and reopen the map editor to continue.'));
  };
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
    return false;
  }
  try {
    const msg = {
      type,
      data,
    };
    const payload = JSON.stringify(msg);

    console.log('Sending to server:', payload);
    socket.send(payload);
    return true;
  } catch (err) {
    console.error('Failed to send WebSocket message:', err);
    return false;
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
