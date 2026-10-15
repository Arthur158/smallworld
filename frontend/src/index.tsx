import React from 'react';
import ReactDOM from 'react-dom/client';
import { I18nextProvider } from 'react-i18next';
import { BrowserRouter } from 'react-router-dom';
import { Provider } from 'react-redux';
import { PersistGate } from 'redux-persist/integration/react';
import { Slide, ToastContainer } from 'react-toastify';
import './index.css';
import TooltipProvider from './components/help/TooltipProvider';
import Router from './routes/Router';
import { WikiProvider } from './components/wiki/WikiContext';
import { persistor, store } from './redux/store';
import i18n from './locales/i18n';

const root = ReactDOM.createRoot(document.getElementById('root') as HTMLElement);
root.render(
  <Provider store={store}>
    <I18nextProvider i18n={i18n}>
      <PersistGate loading={null} persistor={persistor}>
        <BrowserRouter>
          <TooltipProvider><WikiProvider><Router /></WikiProvider></TooltipProvider>
          <ToastContainer
            position="bottom-center"
            autoClose={3000}
            hideProgressBar={false}
            newestOnTop={false}
            closeOnClick
            rtl={false}
            pauseOnFocusLoss
            draggable
            pauseOnHover
            theme="light"
            transition={Slide}
          />
        </BrowserRouter>
      </PersistGate>
    </I18nextProvider>
  </Provider>,
);
