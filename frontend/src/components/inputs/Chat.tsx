import React, { useState } from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '../../redux/store';

export default function Chat() {
  const messages = useSelector((state: RootState) => state.application.messages);
  const [isOpen, setIsOpen] = useState(true);

  const toggleChat = () => {
    setIsOpen((prev) => !prev);
  };

  // Get the last 10 messages in reverse order
  const displayedMessages = messages.slice(-1000).reverse();

  return (
    <div className="w-full h-full relative bg-[#FDF5E6] border border-[#5F4B32] rounded p-0 flex flex-col">

      {isOpen && (
        <div className="mt-8 overflow-auto">
          {displayedMessages.length === 0 ? (
            <p className="text-center italic">Aucun message pour le moment...</p>
          ) : (
            displayedMessages.map((msg, index) => (
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
