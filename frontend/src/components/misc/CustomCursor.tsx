import { useState, useEffect } from "react";
import { RootState } from "../../redux/store";
import { useSelector } from "react-redux";

export default function CustomCursor() {
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const [imageExists, setImageExists] = useState(false);
  const selectedStack = useSelector((state: RootState) => state.application.selectedStack);

  useEffect(() => {
    const updateMousePosition = (e: MouseEvent) => {
      setPosition({ x: e.clientX, y: e.clientY });
    };

    window.addEventListener("mousemove", updateMousePosition);
    return () => window.removeEventListener("mousemove", updateMousePosition);
  }, []);

  useEffect(() => {
    if (!selectedStack) {
      setImageExists(false);
      return;
    }

    const img = new Image();
    img.src = `/stacks/${selectedStack}.png`;

    img.onload = () => setImageExists(true);
    img.onerror = () => setImageExists(false);
  }, [selectedStack]);

  return (
    <>
      {/* Only hide the default cursor when an image exists */}
      {imageExists && (
        <style>
          {`
            * {
              cursor: none;
            }
          `}
        </style>
      )}

      {imageExists && (
        <img
          src={`/stacks/${selectedStack}.png`}
          alt=""
          style={{
            position: "fixed",
            left: position.x,
            top: position.y,
            transform: "translate(-50%, -50%)",
            pointerEvents: "none",
            width: "50px",
            height: "50px",
            zIndex: 9999,
          }}
        />
      )}
    </>
  );
}
