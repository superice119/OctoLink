import PropTypes from 'prop-types';
import {
  Avatar,
  Box,
  Card,
  Checkbox,
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
  TablePagination,
  TextField,
  InputAdornment,
  IconButton,
  Input
} from '@mui/material';
import EyeIcon from '@heroicons/react/24/outline/EyeIcon';
import EyeSlashIcon from '@heroicons/react/24/outline/EyeSlashIcon';
import { Scrollbar } from 'src/components/scrollbar';
import PencilIcon from '@heroicons/react/24/outline/PencilIcon';
import TrashIcon from '@heroicons/react/24/outline/TrashIcon';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

export const CredentialsTable = (props) => {
  const {
    count = 0,
    items = {},
    onDeselectAll,
    onDeselectOne,
    onPageChange = () => {},
    onRowsPerPageChange,
    onSelectAll,
    onSelectOne,
    deleteCredential,
    page = 0,
    rowsPerPage = 0,
    // selected = []
  } = props;

  const [showPassword, setShowPassword] = useState({})
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [credentialToDelete, setCredentialToDelete] = useState("")
  const { t } = useTranslation();

  useEffect(()=>{
    Object.keys(items).map((key) => {
      let newData = {};
      newData[key] = false
      setShowPassword(prevState => ({
        ...prevState,
        ...newData
      }))
    })
    // console.log("showPassword: "+ showPassword)
  },[])
  
  return (
    <Card>
      <Scrollbar>
        <Box sx={{ minWidth: 800 }}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell align='center'>
                  {t('credentials.table.username')}
                </TableCell>
                <TableCell align='center'>
                  {t('credentials.table.password')}
                </TableCell>
                <TableCell align='center'>
                  {t('credentials.table.actions')}
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {Object.keys(items).map((key) => {
                let value = items[key];

                return (
                  <TableRow
                    hover
                    key={key}
                  >
                    <TableCell align='center'>
                      {key}
                    </TableCell>
                    <TableCell align='center'>
                    <Input
                        id="standard-adornment-password"
                        type={showPassword[key] ? 'text' : 'password'}
                        endAdornment={
                          <InputAdornment position="end">
                            <IconButton
                              aria-label="toggle password visibility"
                              onClick={()=>{
                                let newData = {};
                                newData[key] = !showPassword[key]
                                setShowPassword(previous => ({...previous, ...newData}))
                              }}
                              //onMouseDown={handleMouseDownPassword}
                            >
                              <SvgIcon>
                                {showPassword[key] ? <EyeSlashIcon /> : <EyeIcon />}
                              </SvgIcon>
                            </IconButton>
                          </InputAdornment>
                        }
                        value={value}
                      />
                    </TableCell>
                    <TableCell align='center'>
                    <Button
                        onClick={() => {
                          console.log("delete user: ", key)
                          setCredentialToDelete(key);
                          setShowDeleteDialog(true);
                        }}
                      ><SvgIcon
                        color="action"
                        fontSize="small"
                        sx={{ cursor: 'pointer'}}
                      >
                        <TrashIcon
                        ></TrashIcon>
                      </SvgIcon></Button>
                    </TableCell>
                  </TableRow>
                  );
                })
              }
            </TableBody>
          </Table>
        </Box>
      </Scrollbar>
      <Dialog
      open={showDeleteDialog}
      onClose={() => setShowDeleteDialog(false)}
      aria-labelledby="alert-dialog-title"
      aria-describedby="alert-dialog-description"
    >
      <DialogTitle id="alert-dialog-title">{t('credentials.table.deleteTitle')}</DialogTitle>
      <DialogContent>
        <DialogContentText id="alert-dialog-description">
          {t('credentials.table.deleteConfirm')}
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={() => {
          setShowDeleteDialog(false)
          setCredentialToDelete("")
        }} color="primary">
          {t('credentials.table.cancel')}
        </Button>
        <Button onClick={() => {
          deleteCredential(credentialToDelete);
          setShowDeleteDialog(false);
          setCredentialToDelete("")
        }} color="primary" autoFocus>
          {t('credentials.table.delete')}
        </Button>
      </DialogActions>
    </Dialog>
    </Card>
  );
};

CredentialsTable.propTypes = {
  count: PropTypes.number,
  items: PropTypes.object,
  //onDeselectAll: PropTypes.func,
  //onDeselectOne: PropTypes.func,
  onPageChange: PropTypes.func,
  //onRowsPerPageChange: PropTypes.func,
  //onSelectAll: PropTypes.func,
  //onSelectOne: PropTypes.func,
  deleteCredential: PropTypes.func,
  //page: PropTypes.number,
  //rowsPerPage: PropTypes.number,
  //selected: PropTypes.array
};
