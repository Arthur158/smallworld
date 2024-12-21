// Chat.tsx
import React, { useState } from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '../../redux/store';

export default function Chat() {
  const messages = useSelector((state: RootState) => state.application.messages);
  const [isOpen, setIsOpen] = useState(false);

  const toggleChat = () => {
    setIsOpen((prev) => !prev);
  };

  return (
    <div className="w-full h-full relative bg-[#FDF5E6] border border-[#5F4B32] rounded p-2 flex flex-col">
      <button
        className="absolute top-2 right-2 bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded"
        onClick={toggleChat}
      >
        {isOpen ? '▼' : '▲'}
      </button>

      {isOpen && (
        <div className="mt-8 overflow-auto">
          {messages.length === 0 ? (
            <p className="text-center italic">Aucun message pour le moment...</p>
          ) : (
            messages.map((msg, index) => (
              <div key={index} className="mb-2">
                <span className="text-sm">{msg}</span>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
}
