import PropTypes from 'prop-types';
import {
  Avatar,
  Box,
  Card,
  Checkbox,
  Chip,
  Icon,
  Stack,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableHead,
  //TablePagination,
  TableRow,
  Typography,
  SvgIcon,
  Dialog,
  DialogActions,
  DialogTitle,
  DialogContent,
  DialogContentText,
  Button,
  Select,
  MenuItem,
  FormControl,
} from '@mui/material';
import { Scrollbar } from 'src/components/scrollbar';
import { getInitials } from 'src/utils/get-initials';
import TrashIcon from '@heroicons/react/24/outline/TrashIcon';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

const ROLE_COLORS = {
  super_admin: 'error',
  tenant_admin: 'warning',
  operator: 'primary',
  viewer: 'default',
};

export const CustomersTable = (props) => {
  const {
    count = 0,
    items = [],
    onDeselectAll,
    onDeselectOne,
    onPageChange = () => {},
    onRowsPerPageChange,
    onSelectAll,
    onSelectOne,
    deleteUser,
    assignRole,
    canManageRoles = false,
    page = 0,
    rowsPerPage = 0,
    selected = []
  } = props;

  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [userToDelete, setUserToDelete] = useState("")
  const { t } = useTranslation();
  const ROLE_LABELS = {
    super_admin: t('accessControl.users.roles.superAdmin'),
    tenant_admin: t('accessControl.users.roles.tenantAdmin'),
    operator: t('accessControl.users.roles.operator'),
    viewer: t('accessControl.users.roles.viewer'),
  };
  
  return (
    <Card>
      <Scrollbar>
        <Box sx={{ minWidth: 800 }}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell sx={{marginLeft:"30px"}}>
                  {t('customers.table.name')}
                </TableCell>
                <TableCell>
                  {t('customers.table.email')}
                </TableCell>
                <TableCell>
                  {t('customers.table.phone')}
                </TableCell>
                <TableCell>
                  {t('customers.table.role')}
                </TableCell>
                <TableCell>
                  {t('customers.table.tenant')}
                </TableCell>
                <TableCell>
                  {t('customers.table.createdAt')}
                </TableCell>
                <TableCell>
                  {t('customers.table.actions')}
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {items.map((customer) => {
                const isSelected = selected.includes(customer._id);
                const effectiveRole = customer.role || (customer.level == 1 ? 'super_admin' : 'operator');
                return (
                  <TableRow
                    hover
                    key={customer._id}
                    selected={isSelected}
                  >
                    <TableCell align="center" sx={{margin: 'auto', textAlign: 'center'}}>
                      <Stack
                        alignItems="center"
                        direction="row"
                        spacing={2}
                      >
                        <Avatar src={customer.avatar ? customer.avatar : "/assets/avatars/default-avatar.png"}>
                          {getInitials(customer.name)}
                        </Avatar>
                        <Typography variant="subtitle2">
                          {customer.name}
                        </Typography>
                      </Stack>
                    </TableCell>
                    <TableCell>
                      {customer.email}
                    </TableCell>
                    <TableCell>
                      {customer.phone}
                    </TableCell>
                    <TableCell>
                      {canManageRoles ? (
                        <FormControl size="small" variant="standard">
                          <Select
                            value={effectiveRole}
                            onChange={(e) => assignRole && assignRole(customer.email, e.target.value, customer.tenant_id || 'default')}
                          >
                            {['super_admin','tenant_admin','operator','viewer'].map((r) => (
                              <MenuItem key={r} value={r}>{ROLE_LABELS[r] || r}</MenuItem>
                            ))}
                          </Select>
                        </FormControl>
                      ) : (
                        <Chip
                          label={ROLE_LABELS[effectiveRole] || effectiveRole}
                          color={ROLE_COLORS[effectiveRole] || 'default'}
                          size="small"
                        />
                      )}
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" color="text.secondary">
                        {customer.tenant_id || 'default'}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      {customer.createdAt}
                    </TableCell>
                    <TableCell>
                      { customer.level == 0 ? <Button
                        onClick={() => {
                          setUserToDelete(customer.email);
                          setShowDeleteDialog(true);
                        }}
                      ><SvgIcon
                        color="action"
                        fontSize="small"
                        sx={{ cursor: 'pointer'}}
                      >
                        <TrashIcon
                        ></TrashIcon>
                      </SvgIcon></Button>: <span></span>}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </Box>
      </Scrollbar>
      {/* <TablePagination
        component="div"
        count={count}
        //onPageChange={onPageChange}
        //onRowsPerPageChange={onRowsPerPageChange}
        //page={page}
        //rowsPerPage={rowsPerPage}
        //rowsPerPageOptions={[5, 10, 25]}
      /> */}
      <Dialog
      open={showDeleteDialog}
      onClose={() => setShowDeleteDialog(false)}
      aria-labelledby="alert-dialog-title"
      aria-describedby="alert-dialog-description"
    >
      <DialogTitle id="alert-dialog-title">{t('customers.table.deleteTitle')}</DialogTitle>
      <DialogContent>
        <DialogContentText id="alert-dialog-description">
          {t('customers.table.deleteConfirm')}
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={() => {
          setShowDeleteDialog(false)
          setUserToDelete("")
        }} color="primary">
          {t('customers.table.cancel')}
        </Button>
        <Button onClick={() => {
          deleteUser(userToDelete);
          setShowDeleteDialog(false);
          setUserToDelete("")
        }} color="primary" autoFocus>
          {t('customers.table.delete')}
        </Button>
      </DialogActions>
    </Dialog>
    </Card>
  );
};

CustomersTable.propTypes = {
  count: PropTypes.number,
  items: PropTypes.array,
  onDeselectAll: PropTypes.func,
  onDeselectOne: PropTypes.func,
  onPageChange: PropTypes.func,
  //onRowsPerPageChange: PropTypes.func,
  onSelectAll: PropTypes.func,
  onSelectOne: PropTypes.func,
  deleteUser: PropTypes.func,
  //page: PropTypes.number,
  //rowsPerPage: PropTypes.number,
  selected: PropTypes.array
};
