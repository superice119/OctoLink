import Head from 'next/head';
import { CacheProvider } from '@emotion/react';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { CssBaseline } from '@mui/material';
import { ThemeProvider } from '@mui/material/styles';
import { AuthConsumer, AuthProvider } from 'src/contexts/auth-context';
import { useNProgress } from 'src/hooks/use-nprogress';
import { createTheme } from 'src/theme';
import { createEmotionCache } from 'src/utils/create-emotion-cache';
import 'simplebar-react/dist/simplebar.min.css';
import '../utils/map.css';
import { useEffect, useState } from 'react';
import { I18nextProvider } from 'react-i18next';
import { BackendProvider } from 'src/contexts/backend-context';
import { AlertProvider } from 'src/contexts/error-context';
import { NotificationProvider } from 'src/contexts/notification-context';
import i18n from 'src/i18n';

const clientSideEmotionCache = createEmotionCache();

const SplashScreen = () => null;

const App = (props) => {
  const [theme, setTheme] = useState(null);
  const { Component, emotionCache = clientSideEmotionCache, pageProps } = props;

  useNProgress();

  const getLayout = Component.getLayout ?? ((page) => page);

  useEffect(() => {
    setTheme(createTheme());
    // Keep <html lang> in sync with the active language for a11y/SEO.
    if (typeof document !== 'undefined') {
      const apply = (lng) => {
        document.documentElement.lang = (lng || 'zh').split('-')[0];
      };
      apply(i18n.resolvedLanguage || i18n.language);
      i18n.on('languageChanged', apply);
      return () => i18n.off('languageChanged', apply);
    }
  }, []);

  return theme && (
    <CacheProvider value={emotionCache}>
      <I18nextProvider i18n={i18n}>
      <Head>
        <title>
          OctoLink | 物联控制器
        </title>
        <meta
          name="viewport"
          content="initial-scale=1, width=device-width"
        />
      </Head>
      <LocalizationProvider dateAdapter={AdapterDateFns}>
        <AuthProvider>
          {/* <WsProvider> */}
          <AlertProvider>
            <BackendProvider>
              <NotificationProvider>
                <ThemeProvider theme={theme}>
                  <CssBaseline />
                  <AuthConsumer>
                    {
                      (auth) => auth.isLoading
                        ? <SplashScreen />
                        : getLayout(<Component {...pageProps} />)
                    }
                  </AuthConsumer>
                </ThemeProvider>
              </NotificationProvider>
            </BackendProvider>
          </AlertProvider>
          {/* </WsProvider> */}
        </AuthProvider>
      </LocalizationProvider>
      </I18nextProvider>
    </CacheProvider>
  );
};

export default App;
