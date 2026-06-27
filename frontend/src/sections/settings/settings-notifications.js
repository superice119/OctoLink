import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  CardActions,
  CardContent,
  CardHeader,
  Checkbox,
  Divider,
  FormControlLabel,
  Stack,
  Typography,
  Unstable_Grid2 as Grid
} from '@mui/material';

export const SettingsNotifications = () => {
  const { t } = useTranslation();
  const handleSubmit = useCallback(
    (event) => {
      event.preventDefault();
    },
    []
  );

  return (
    <form onSubmit={handleSubmit}>
      <Card>
        <CardHeader
          subheader={t('settings.notifications.manageSubheader')}
          title={t('settings.notifications.title')}
        />
        <Divider />
        <CardContent>
          <Grid
            container
            spacing={6}
            wrap="wrap"
          >
            <Grid
              xs={12}
              sm={6}
              md={4}
            >
              <Stack spacing={1}>
                <Typography variant="h6">
                  {t('settings.notifications.notificationsGroup')}
                </Typography>
                <Stack>
                  <FormControlLabel
                    control={<Checkbox defaultChecked />}
                    label={t('settings.notifications.email')}
                  />
                  <FormControlLabel
                    control={<Checkbox defaultChecked />}
                    label={t('settings.notifications.pushNotifications')}
                  />
                  <FormControlLabel
                    control={<Checkbox />}
                    label={t('settings.notifications.textMessages')}
                  />
                  <FormControlLabel
                    control={<Checkbox defaultChecked />}
                    label={t('settings.notifications.phoneCalls')}
                  />
                </Stack>
              </Stack>
            </Grid>
            <Grid
              item
              md={4}
              sm={6}
              xs={12}
            >
              <Stack spacing={1}>
                <Typography variant="h6">
                  {t('settings.notifications.messagesGroup')}
                </Typography>
                <Stack>
                  <FormControlLabel
                    control={<Checkbox defaultChecked />}
                    label={t('settings.notifications.email')}
                  />
                  <FormControlLabel
                    control={<Checkbox />}
                    label={t('settings.notifications.pushNotifications')}
                  />
                  <FormControlLabel
                    control={<Checkbox defaultChecked />}
                    label={t('settings.notifications.phoneCalls')}
                  />
                </Stack>
              </Stack>
            </Grid>
          </Grid>
        </CardContent>
        <Divider />
        <CardActions sx={{ justifyContent: 'flex-end' }}>
          <Button variant="contained">
            {t('settings.notifications.save')}
          </Button>
        </CardActions>
      </Card>
    </form>
  );
};
