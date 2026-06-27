import {
  Box, Card, CardHeader, Chip, Divider, IconButton, List, ListItem,
  ListItemText, Tooltip, Typography, SvgIcon
} from '@mui/material';
import TrashIcon from '@heroicons/react/24/outline/TrashIcon';
import CheckCircleIcon from '@heroicons/react/24/outline/CheckCircleIcon';
import { format } from 'date-fns';
import { useNotifications } from 'src/contexts/notification-context';
import { useTranslation } from 'react-i18next';

const TYPE_COLORS = {
  event: 'primary',
  value_change: 'info',
  obj_creation: 'success',
  obj_deletion: 'error',
  oper_complete: 'warning',
  on_board_req: 'secondary',
  unknown: 'default',
};

export const NotificationsList = () => {
  const { items, unread, markAllRead, clearAll } = useNotifications();
  const { t } = useTranslation();

  return (
    <Card>
      <CardHeader
        title={(
          <Box display="flex" alignItems="center" gap={1}>
            {t('notifications.list.title')}
            {unread > 0 && <Chip label={unread} color="error" size="small" />}
          </Box>
        )}
        action={(
          <Box>
            <Tooltip title={t('notifications.list.markAllRead')}>
              <IconButton onClick={markAllRead}>
                <SvgIcon fontSize="small"><CheckCircleIcon /></SvgIcon>
              </IconButton>
            </Tooltip>
            <Tooltip title={t('notifications.list.clearAll')}>
              <IconButton onClick={clearAll}>
                <SvgIcon fontSize="small"><TrashIcon /></SvgIcon>
              </IconButton>
            </Tooltip>
          </Box>
        )}
      />
      <Divider />
      <List disablePadding>
        {items.length === 0 && (
          <Box sx={{ p: 3, textAlign: 'center' }}>
            <Typography variant="body2" color="text.secondary">{t('notifications.list.empty')}</Typography>
          </Box>
        )}
        {items.map((n, idx) => (
          <ListItem
            key={n.id || idx}
            divider
            sx={{ bgcolor: n.read ? 'transparent' : 'action.hover' }}
          >
            <ListItemText
              primary={(
                <Box display="flex" alignItems="center" gap={1}>
                  <Chip
                    label={n.type || 'event'}
                    color={TYPE_COLORS[n.type] || 'default'}
                    size="small"
                  />
                  <Typography variant="body2" noWrap>
                    {n.event_name || n.obj_path || n.param_path || n.device_sn}
                  </Typography>
                </Box>
              )}
              secondary={(
                <Box>
                  <Typography variant="caption" color="text.secondary">
                    {t('notifications.list.device')}: {n.device_sn}
                  </Typography>
                  {n.param_value && (
                    <Typography variant="caption" color="text.secondary" sx={{ ml: 1 }}>
                      {t('notifications.list.value')}: {n.param_value}
                    </Typography>
                  )}
                  <Typography variant="caption" color="text.secondary" sx={{ ml: 1 }}>
                    {n.timestamp ? format(new Date(n.timestamp), 'yyyy-MM-dd HH:mm:ss') : ''}
                  </Typography>
                </Box>
              )}
            />
          </ListItem>
        ))}
      </List>
    </Card>
  );
};
