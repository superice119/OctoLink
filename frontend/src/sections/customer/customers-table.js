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
  
  return (
    <Card>
      <Scrollbar>
        <Box sx={{ minWidth: 800 }}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell sx={{marginLeft:"30px"}}>
                  Name
                </TableCell>
                <TableCell>
                  Email
                </TableCell>
                <TableCell>
                  Phone
                </TableCell>
                <TableCell>
                  Role
                </TableCell>
                <TableCell>
                  Tenant
                </TableCell>
                <TableCell>
                  Created At
                </TableCell>
                <TableCell>
                  Actions
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
                              <MenuItem key={r} value={r}>{r}</MenuItem>
                            ))}
                          </Select>
                        </FormControl>
                      ) : (
                        <Chip
                          label={effectiveRole}
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
      <DialogTitle id="alert-dialog-title">{"Delete User"}</DialogTitle>
      <DialogContent>
        <DialogContentText id="alert-dialog-description">
          Are you sure you want to delete this user?
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={() => {
          setShowDeleteDialog(false)
          setUserToDelete("")
        }} color="primary">
          Cancel
        </Button>
        <Button onClick={() => {
          deleteUser(userToDelete);
          setShowDeleteDialog(false);
          setUserToDelete("")
        }} color="primary" autoFocus>
          Delete
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
