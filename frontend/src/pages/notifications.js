import Head from 'next/head';
import { Box, Container, Typography } from '@mui/material';
import { Layout as DashboardLayout } from 'src/layouts/dashboard/layout';
import { NotificationsList } from 'src/sections/notifications/notifications-list';

const NotificationsPage = () => {
  return (
    <>
      <Head>
        <title>通知 | OctoLink</title>
      </Head>
      <Box
        component="main"
        sx={{ flexGrow: 1, py: 8 }}
      >
        <Container maxWidth="xl">
          <Typography variant="h4" sx={{ mb: 3 }}>通知中心</Typography>
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
