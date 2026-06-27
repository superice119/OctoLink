import { useCallback, useState, useEffect } from 'react';
import Head from 'next/head';
import { useTranslation } from 'react-i18next';
import PlusIcon from '@heroicons/react/24/solid/PlusIcon';
import TrashIcon from '@heroicons/react/24/solid/TrashIcon';
import ShieldCheckIcon from '@heroicons/react/24/solid/ShieldCheckIcon';
import {
  Box,
  Button,
  Chip,
  Container,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  FormGroup,
  FormControlLabel,
  Checkbox,
  IconButton,
  Stack,
  SvgIcon,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Typography,
  Paper,
  Alert,
} from '@mui/material';
import { Layout as DashboardLayout } from 'src/layouts/dashboard/layout';
import { useAuth } from 'src/hooks/use-auth';
import { useRouter } from 'next/router';

const ALL_PERMISSIONS = [
  'devices:read',
  'devices:write',
  'users:read',
  'users:write',
  'tenants:manage',
  'roles:manage',
];

const SYSTEM_ROLES = ['super_admin', 'tenant_admin', 'operator', 'viewer'];

const Page = () => {
  const auth = useAuth();
  const router = useRouter();
  const { t } = useTranslation();

  const [roles, setRoles] = useState([]);
  const [loading, setLoading] = useState(true);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [newRoleName, setNewRoleName] = useState('');
  const [newRolePerms, setNewRolePerms] = useState([]);
  const [error, setError] = useState('');

  const userRole = auth.user?.role || '';
  const canManage = userRole === 'super_admin' || userRole === 'tenant_admin';

  const fetchRoles = async () => {
    setLoading(true);
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_REST_ENDPOINT || ''}/api/roles`, {
        headers: { Authorization: auth.user?.token },
      });
      if (res.status === 401) return router.push('/auth/login');
      if (res.status === 403) return router.push('/403');
      const data = await res.json();
      setRoles(data || []);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchRoles(); }, []);

  const handlePermToggle = (perm) => {
    setNewRolePerms((prev) =>
      prev.includes(perm) ? prev.filter((p) => p !== perm) : [...prev, perm]
    );
  };

  const handleCreate = async () => {
    if (!newRoleName.trim()) { setError(t('accessControl.roles.roleNameRequired')); return; }
    setError('');
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_REST_ENDPOINT || ''}/api/roles`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: auth.user?.token },
        body: JSON.stringify({ name: newRoleName.trim(), permissions: newRolePerms }),
      });
      if (res.status === 401) return router.push('/auth/login');
      if (res.status === 403) { setError(t('accessControl.roles.insufficientPermissions')); return; }
      if (res.status === 409) { setError(t('accessControl.roles.roleNameExists')); return; }
      if (!res.ok) { setError(t('accessControl.roles.failedToCreateRole')); return; }
      setDialogOpen(false);
      setNewRoleName('');
      setNewRolePerms([]);
      fetchRoles();
    } catch (e) {
      setError(t('accessControl.roles.networkError'));
    }
  };

  const handleDelete = async (id) => {
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_REST_ENDPOINT || ''}/api/roles/${id}`, {
        method: 'DELETE',
        headers: { Authorization: auth.user?.token },
      });
      if (res.status === 401) return router.push('/auth/login');
      if (res.status === 403) return router.push('/403');
      fetchRoles();
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <>
      <Head><title>{t('accessControl.roles.headTitle')}</title></Head>
      <Box component="main" sx={{ flexGrow: 1, py: 8 }}>
        <Container maxWidth="xl">
          <Stack spacing={3}>
            <Stack direction="row" justifyContent="space-between" alignItems="center">
              <Typography variant="h4">{t('accessControl.roles.pageTitle')}</Typography>
              {canManage && (
                <Button
                  startIcon={<SvgIcon fontSize="small"><PlusIcon /></SvgIcon>}
                  variant="contained"
                  onClick={() => { setDialogOpen(true); setError(''); }}
                >
                  {t('accessControl.roles.addCustomRole')}
                </Button>
              )}
            </Stack>

            <Paper>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>{t('accessControl.roles.name')}</TableCell>
                    <TableCell>{t('accessControl.roles.type')}</TableCell>
                    <TableCell>{t('accessControl.roles.permissions')}</TableCell>
                    {canManage && <TableCell align="right">{t('accessControl.roles.actions')}</TableCell>}
                  </TableRow>
                </TableHead>
                <TableBody>
                  {roles.map((role) => (
                    <TableRow key={role.id} hover>
                      <TableCell>
                        <Stack direction="row" alignItems="center" spacing={1}>
                          <SvgIcon fontSize="small" color="primary">
                            <ShieldCheckIcon />
                          </SvgIcon>
                          <Typography variant="body2">{role.name}</Typography>
                        </Stack>
                      </TableCell>
                      <TableCell>
                        <Chip
                          label={role.is_system ? t('accessControl.roles.system') : t('accessControl.roles.custom')}
                          color={role.is_system ? 'default' : 'primary'}
                          size="small"
                        />
                      </TableCell>
                      <TableCell>
                        <Stack direction="row" flexWrap="wrap" gap={0.5}>
                          {(role.permissions || []).map((p) => (
                            <Chip key={p} label={p} size="small" variant="outlined" />
                          ))}
                        </Stack>
                      </TableCell>
                      {canManage && (
                        <TableCell align="right">
                          {!role.is_system && (
                            <IconButton
                              size="small"
                              color="error"
                              onClick={() => handleDelete(role.id)}
                            >
                              <SvgIcon fontSize="small"><TrashIcon /></SvgIcon>
                            </IconButton>
                          )}
                        </TableCell>
                      )}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </Paper>
          </Stack>
        </Container>
      </Box>

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{t('accessControl.roles.createCustomRole')}</DialogTitle>
        <DialogContent>
          {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
          <TextField
            autoFocus
            fullWidth
            label={t('accessControl.roles.roleName')}
            value={newRoleName}
            onChange={(e) => setNewRoleName(e.target.value)}
            margin="normal"
            variant="standard"
          />
          <Typography variant="subtitle2" sx={{ mt: 2, mb: 1 }}>{t('accessControl.roles.permissions')}</Typography>
          <FormControl component="fieldset">
            <FormGroup>
              {ALL_PERMISSIONS.map((perm) => (
                <FormControlLabel
                  key={perm}
                  control={
                    <Checkbox
                      checked={newRolePerms.includes(perm)}
                      onChange={() => handlePermToggle(perm)}
                    />
                  }
                  label={perm}
                />
              ))}
            </FormGroup>
          </FormControl>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => { setDialogOpen(false); setError(''); }}>{t('accessControl.roles.cancel')}</Button>
          <Button variant="contained" onClick={handleCreate}>{t('accessControl.roles.create')}</Button>
        </DialogActions>
      </Dialog>
    </>
  );
};

Page.getLayout = (page) => <DashboardLayout>{page}</DashboardLayout>;

export default Page;
