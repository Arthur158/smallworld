// import React, { createContext, useEffect, useContext } from 'react';
// import { connectWebSocket} from '../services/backendService'
// import { useDispatch } from 'react-redux';
// import { handleWebSocketCallback } from '../redux/slices/applicationSlice';
// import { AppDispatch } from '../redux/store';
//
// const WebSocketContext = createContext<WebSocket | null>(null);
//
// export const WebSocketProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
//   const dispatch: AppDispatch = useDispatch();
//
//   useEffect(() => {
//     // Connect WebSocket
//     connectWebSocket('ws://yourserver.com');
//
//     // Attach Redux-connected callback
//     onWebSocketMessage((message: any) => {
//       dispatch(handleWebSocketCallback(message));
//     });
//
//     // Optional: Cleanup WebSocket connection
//     return () => {
//       console.log('WebSocket disconnected.');
//     };
//   }, [dispatch]);
//
//   return (
//     <WebSocketContext.Provider value={null}>
//       {children}
//     </WebSocketContext.Provider>
//   );
// };
//
// export const useWebSocket = () => {
//   return useContext(WebSocketContext);
// };
