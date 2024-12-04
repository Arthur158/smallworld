import { combineReducers } from 'redux';
import applicationReducer from './slices/applicationSlice';

const rootReducer = combineReducers({
  application: applicationReducer,
});

export default rootReducer;
