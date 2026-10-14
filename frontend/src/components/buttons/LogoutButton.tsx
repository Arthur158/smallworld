import { useDispatch } from "react-redux";
import { getAuthConfig, sendMessageToBackend } from '../../services/backendService';
import { reset } from '../../redux/slices/applicationSlice';

const LogoutButton = () => {
  const dispatch = useDispatch();

  const handleLogout = async () => {
    const authConfig = await getAuthConfig().catch(() => ({ enabled: false }));
    if (authConfig.enabled) {
      dispatch(reset());
      window.location.assign('/auth/logout');
      return;
    }
    sendMessageToBackend('logout', {});
  };

  return <button onClick={handleLogout}>Logout</button>;
};

export default LogoutButton;
