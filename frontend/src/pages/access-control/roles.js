import { useCallback, useState, useEffect } from 'react';
import Head from 'next/head';
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
    if (!newRoleName.trim()) { setError('Role name is required'); return; }
    setError('');
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_REST_ENDPOINT || ''}/api/roles`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: auth.user?.token },
        body: JSON.stringify({ name: newRoleName.trim(), permissions: newRolePerms }),
      });
      if (res.status === 401) return router.push('/auth/login');
      if (res.status === 403) { setError('Insufficient permissions'); return; }
      if (res.status === 409) { setError('Role name already exists'); return; }
      if (!res.ok) { setError('Failed to create role'); return; }
      setDialogOpen(false);
      setNewRoleName('');
      setNewRolePerms([]);
      fetchRoles();
    } catch (e) {
      setError('Network error');
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
      <Head><title>OctoLink | Roles</title></Head>
      <Box component="main" sx={{ flexGrow: 1, py: 8 }}>
        <Container maxWidth="xl">
          <Stack spacing={3}>
            <Stack direction="row" justifyContent="space-between" alignItems="center">
              <Typography variant="h4">Roles</Typography>
              {canManage && (
                <Button
                  startIcon={<SvgIcon fontSize="small"><PlusIcon /></SvgIcon>}
                  variant="contained"
                  onClick={() => { setDialogOpen(true); setError(''); }}
                >
                  Add Custom Role
                </Button>
              )}
            </Stack>

            <Paper>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>Name</TableCell>
                    <TableCell>Type</TableCell>
                    <TableCell>Permissions</TableCell>
                    {canManage && <TableCell align="right">Actions</TableCell>}
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
                          label={role.is_system ? 'System' : 'Custom'}
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
        <DialogTitle>Create Custom Role</DialogTitle>
        <DialogContent>
          {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
          <TextField
            autoFocus
            fullWidth
            label="Role Name"
            value={newRoleName}
            onChange={(e) => setNewRoleName(e.target.value)}
            margin="normal"
            variant="standard"
          />
          <Typography variant="subtitle2" sx={{ mt: 2, mb: 1 }}>Permissions</Typography>
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
          <Button onClick={() => { setDialogOpen(false); setError(''); }}>Cancel</Button>
          <Button variant="contained" onClick={handleCreate}>Create</Button>
        </DialogActions>
      </Dialog>
    </>
  );
};

Page.getLayout = (page) => <DashboardLayout>{page}</DashboardLayout>;

export default Page;
