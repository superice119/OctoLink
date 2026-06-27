import { useState, useEffect } from 'react';
import Head from 'next/head';
import { useTranslation } from 'react-i18next';
import PlusIcon from '@heroicons/react/24/solid/PlusIcon';
import TrashIcon from '@heroicons/react/24/solid/TrashIcon';
import BuildingOffice2Icon from '@heroicons/react/24/solid/BuildingOffice2Icon';
import {
  Alert,
  Box,
  Button,
  Container,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  Paper,
  Stack,
  SvgIcon,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material';
import { Layout as DashboardLayout } from 'src/layouts/dashboard/layout';
import { useAuth } from 'src/hooks/use-auth';
import { useRouter } from 'next/router';

const Page = () => {
  const auth = useAuth();
  const router = useRouter();
  const { t } = useTranslation();

  const [tenants, setTenants] = useState([]);
  const [loading, setLoading] = useState(true);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [newTenant, setNewTenant] = useState({ name: '', description: '' });
  const [error, setError] = useState('');

  const userRole = auth.user?.role || '';
  if (userRole !== 'super_admin') {
    if (typeof window !== 'undefined') router.push('/403');
    return null;
  }

  const fetchTenants = async () => {
    setLoading(true);
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_REST_ENDPOINT || ''}/api/tenants`, {
        headers: { Authorization: auth.user?.token },
      });
      if (res.status === 401) return router.push('/auth/login');
      if (res.status === 403) return router.push('/403');
      const data = await res.json();
      setTenants(data || []);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchTenants(); }, []);

  const handleCreate = async () => {
    if (!newTenant.name.trim()) { setError(t('accessControl.tenants.tenantNameRequired')); return; }
    setError('');
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_REST_ENDPOINT || ''}/api/tenants`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: auth.user?.token },
        body: JSON.stringify(newTenant),
      });
      if (res.status === 401) return router.push('/auth/login');
      if (res.status === 409) { setError(t('accessControl.tenants.tenantExists')); return; }
      if (!res.ok) { setError(t('accessControl.tenants.failedToCreateTenant')); return; }
      setDialogOpen(false);
      setNewTenant({ name: '', description: '' });
      fetchTenants();
    } catch (e) {
      setError(t('accessControl.tenants.networkError'));
    }
  };

  const handleDelete = async (id) => {
    if (id === 'default') { setError(t('accessControl.tenants.cannotDeleteDefaultTenant')); return; }
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_REST_ENDPOINT || ''}/api/tenants/${id}`, {
        method: 'DELETE',
        headers: { Authorization: auth.user?.token },
      });
      if (res.status === 401) return router.push('/auth/login');
      fetchTenants();
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <>
      <Head><title>{t('accessControl.tenants.headTitle')}</title></Head>
      <Box component="main" sx={{ flexGrow: 1, py: 8 }}>
        <Container maxWidth="xl">
          <Stack spacing={3}>
            <Stack direction="row" justifyContent="space-between" alignItems="center">
              <Typography variant="h4">{t('accessControl.tenants.pageTitle')}</Typography>
              <Button
                startIcon={<SvgIcon fontSize="small"><PlusIcon /></SvgIcon>}
                variant="contained"
                onClick={() => { setDialogOpen(true); setError(''); }}
              >
                {t('accessControl.tenants.addTenant')}
              </Button>
            </Stack>

            {error && <Alert severity="error">{error}</Alert>}

            <Paper>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>{t('accessControl.tenants.id')}</TableCell>
                    <TableCell>{t('accessControl.tenants.name')}</TableCell>
                    <TableCell>{t('accessControl.tenants.description')}</TableCell>
                    <TableCell align="right">{t('accessControl.tenants.actions')}</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {tenants.map((tenant) => (
                    <TableRow key={tenant.id} hover>
                      <TableCell>
                        <Typography variant="body2" color="text.secondary">{tenant.id}</Typography>
                      </TableCell>
                      <TableCell>
                        <Stack direction="row" alignItems="center" spacing={1}>
                          <SvgIcon fontSize="small" color="primary">
                            <BuildingOffice2Icon />
                          </SvgIcon>
                          <Typography variant="body2">{tenant.name}</Typography>
                        </Stack>
                      </TableCell>
                      <TableCell>{tenant.description}</TableCell>
                      <TableCell align="right">
                        {tenant.id !== 'default' && (
                          <IconButton
                            size="small"
                            color="error"
                            onClick={() => handleDelete(tenant.id)}
                          >
                            <SvgIcon fontSize="small"><TrashIcon /></SvgIcon>
                          </IconButton>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </Paper>
          </Stack>
        </Container>
      </Box>

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{t('accessControl.tenants.createTenant')}</DialogTitle>
        <DialogContent>
          {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
          <TextField
            autoFocus
            fullWidth
            label={t('accessControl.tenants.name')}
            value={newTenant.name}
            onChange={(e) => setNewTenant({ ...newTenant, name: e.target.value })}
            margin="normal"
            variant="standard"
          />
          <TextField
            fullWidth
            label={t('accessControl.tenants.description')}
            value={newTenant.description}
            onChange={(e) => setNewTenant({ ...newTenant, description: e.target.value })}
            margin="normal"
            variant="standard"
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => { setDialogOpen(false); setError(''); }}>{t('accessControl.tenants.cancel')}</Button>
          <Button variant="contained" onClick={handleCreate}>{t('accessControl.tenants.create')}</Button>
        </DialogActions>
      </Dialog>
    </>
  );
};

Page.getLayout = (page) => <DashboardLayout>{page}</DashboardLayout>;

export default Page;
