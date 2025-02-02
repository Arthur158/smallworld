import { useDispatch } from "react-redux";
import { useNavigate } from "react-router-dom";
import { sendMessageToBackend } from '../../services/backendService';

const LogoutButton = () => {
  const dispatch = useDispatch();
  const navigate = useNavigate();

  const handleLogout = () => {
    sendMessageToBackend('logout', {})
    navigate("/", { replace: true }); // Redirect to login
  };

  return <button onClick={handleLogout}>Logout</button>;
};

export default LogoutButton;
