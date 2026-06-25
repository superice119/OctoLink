import Head from 'next/head';
import { useTranslation } from 'react-i18next';
import { Box, Container, Stack, Typography } from '@mui/material';
import { SettingsNotifications } from 'src/sections/settings/settings-notifications';
import { SettingsPassword } from 'src/sections/settings/settings-password';
import { Layout as DashboardLayout } from 'src/layouts/dashboard/layout';

const Page = () => {
  const { t } = useTranslation();
  return (
  <>
    <Head>
      <title>
        {t('settings.headTitle')}
      </title>
    </Head>
    <Box
      component="main"
      sx={{
        flexGrow: 1,
        py: 8
      }}
    >
      <Container maxWidth="lg">
        <Stack spacing={3}>
          <Typography variant="h4">
            {t('settings.title')}
          </Typography>
          {/*<SettingsNotifications />*/}
          <SettingsPassword />
        </Stack>
      </Container>
    </Box>
  </>
  );
};

Page.getLayout = (page) => (
  <DashboardLayout>
    {page}
  </DashboardLayout>
);

export default Page;
