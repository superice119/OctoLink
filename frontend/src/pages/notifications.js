import Head from 'next/head';
import { Box, Container, Typography } from '@mui/material';
import { useTranslation } from 'react-i18next';
import { Layout as DashboardLayout } from 'src/layouts/dashboard/layout';
import { NotificationsList } from 'src/sections/notifications/notifications-list';

const NotificationsPage = () => {
  const { t } = useTranslation();
  return (
    <>
      <Head>
        <title>{t('notifications.headTitle')}</title>
      </Head>
      <Box
        component="main"
        sx={{ flexGrow: 1, py: 8 }}
      >
        <Container maxWidth="xl">
          <Typography variant="h4" sx={{ mb: 3 }}>{t('notifications.title')}</Typography>
          <NotificationsList />
        </Container>
      </Box>
    </>
  );
};

NotificationsPage.getLayout = (page) => (
  <DashboardLayout>{page}</DashboardLayout>
);

export default NotificationsPage;
