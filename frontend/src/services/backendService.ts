import { Polygon } from '../types/Board';
import { websocketMessageReceived } from '../redux/slices/applicationSlice';
import { store } from '../redux/store'; // Import the Redux store
import { useSelector, useDispatch } from 'react-redux';
import { RootState, AppDispatch } from '../redux/store';

let socket: WebSocket | null = null;


// Connect WebSocket
export const connectWebSocket = () => {
  socket = new WebSocket('wss://6574-2a02-2788-11ca-f01-2e0-4cff-fe68-ed.ngrok-free.app/ws');
  // 'https://397a-2a02-2788-11ca-f01-2e0-4cff-fe68-ed.ngrok-free.app/' 
  // -> 'wss://397a-2a02-2788-11ca-f01-2e0-4cff-fe68-ed.ngrok-free.app/ws'
  // ws://localhost:8080/ws

  socket.onopen = () => console.log('WebSocket connected');
  socket.onclose = () => console.log('WebSocket disconnected');
  socket.onerror = (err) => console.error('WebSocket error:', err);

  // Single onmessage handler
  socket.onmessage = (event) => {
    const message = event.data
    console.log(message)
    store.dispatch(websocketMessageReceived(message));
  };
};


export const sendMessageToBackend = (type: string, data: Record<string, any>) => {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    console.error('WebSocket is not open.');
    return;
  }


  try {
    // Construct the message with Type and Data fields
    const message = {
      type: type,
      data: data,
    };

    // Convert the message to a JSON string
    const payload = JSON.stringify(message);

    // Send the message to the backend
    console.log("abt to send...")
    console.log(payload)
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
