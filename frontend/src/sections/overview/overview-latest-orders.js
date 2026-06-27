import { format } from 'date-fns';
import PropTypes from 'prop-types';
import { useTranslation } from 'react-i18next';
import ArrowRightIcon from '@heroicons/react/24/solid/ArrowRightIcon';
import ArrowTopRightOnSquareIcon from '@heroicons/react/24/solid/ArrowTopRightOnSquareIcon';
import {
  Box,
  Button,
  Card,
  CardActions,
  CardHeader,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Input,
  InputLabel,
  SvgIcon,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Tooltip
} from '@mui/material';
import { Scrollbar } from 'src/components/scrollbar';
import { SeverityPill } from 'src/components/severity-pill';
import { useRouter } from 'next/router';
import { useState } from 'react';
import PencilIcon from '@heroicons/react/24/outline/PencilIcon';

const statusMap = {
  1: 'warning',
  2: 'success',
  0: 'error'
};

const statusKey = (s)=>{
  if (s == 0){
    return "statusOffline"
  } else if (s == 1){
    return "statusAssociating"
  }else if (s==2){
    return "statusOnline"
  }else {
    return "statusUnknown"
  }
}

const getDeviceProtocol = (order) => {
  if (order.Mqtt == 0 && order.Websockets == 0 && order.Stomp == 0) {
    return "cwmp"
  }else {
    return "usp"
  }
}

export const OverviewLatestOrders = (props) => {
  const { orders = [], sx } = props;

  const router = useRouter()
  const { t } = useTranslation();

  const [showSetDeviceAlias, setShowSetDeviceAlias] = useState(false);
  const [deviceAlias, setDeviceAlias] = useState(null);
  const [deviceToBeChanged, setDeviceToBeChanged] = useState(null);

  const setNewDeviceAlias = async (alias,sn) => {
    var myHeaders = new Headers();
    myHeaders.append("Content-Type", "application/json");
    myHeaders.append("Authorization", localStorage.getItem("token"));

    var requestOptions = {
      method: 'PUT',
      headers: myHeaders,
      body: alias,
      redirect: 'follow'
    };

    let result = await fetch(`${process.env.NEXT_PUBLIC_REST_ENDPOINT || ""}/api/device/alias?id=${sn}`, requestOptions)
    console.log("result:", result)
    if (result.status === 401){
      router.push("/auth/login")
    }else if (result.status != 200){
      console.log("Status:", result.status)
      let content = await result.json()
      console.log("Message:", content)
      setShowSetDeviceAlias(false)
      setDeviceAlias(null)
      setDeviceToBeChanged(null)
    }else{
      let content = await result.json()
      console.log("set alias result:", content)
      setShowSetDeviceAlias(false)
      setDeviceAlias(null)
      orders[deviceToBeChanged].Alias = alias
      setDeviceToBeChanged(null)
    }
    // .then(response => {
    //   if (response.status === 401) {
    //     router.push("/auth/login")
    //   }
    //   return response.json()
    // })
    // .then(result => {
    //   console.log("alias result:", result)
    //   setShowSetDeviceAlias(false)
    //   setDeviceAlias(null)
    // })
    // .catch(error => {
    //   console.log('error:', error)
    //   setShowSetDeviceAlias(false)
    //   setDeviceAlias(null)
    // })
  }

  return (<div>
    <Card sx={sx}>
      <CardHeader title={t('overview.latestOrders.title')} />
      <Scrollbar sx={{ flexGrow: 1 }}>
        <Box sx={{ minWidth: 800 }}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell align="center">
                  {t('overview.latestOrders.serialNumber')}
                </TableCell>
                <TableCell>
                  {t('overview.latestOrders.alias')}
                </TableCell>
                <TableCell>
                  {t('overview.latestOrders.model')}
                </TableCell>
                <TableCell>
                  {t('overview.latestOrders.vendor')}
                </TableCell>
                <TableCell>
                  {t('overview.latestOrders.version')}
                </TableCell>
                <TableCell>
                  {t('overview.latestOrders.status')}
                </TableCell>
                <TableCell align="center">
                  {t('overview.latestOrders.actions')}
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {orders && orders.map((order, index) => {

                return (
                  <TableRow
                    hover
                    key={order.SN}
                  >
                    <TableCell align="center">
                      {order.SN}
                    </TableCell>
                    <TableCell>
                      {order.Alias}
                    </TableCell>
                    <TableCell>
                      {order.Model || order.ProductClass}
                    </TableCell>
                    <TableCell>
                      {order.Vendor}
                    </TableCell>
                    <TableCell>
                      {order.Version}
                    </TableCell>
                    <TableCell>
                    <SeverityPill color={statusMap[order.Status]}>
                        {t('overview.latestOrders.' + statusKey(order.Status))}
                    </SeverityPill>
                    </TableCell>
                    <TableCell align="center">
                    {order.Status == 2 && 
                      <Tooltip title={t('overview.latestOrders.accessDevice')}>
                        <Button
                          onClick={()=>{
                            if (getDeviceProtocol(order) == "usp"){
                              router.push("devices/"+ getDeviceProtocol(order) +"/"+order.SN+"/discovery")
                            }else {
                              router.push("devices/"+ getDeviceProtocol(order) +"/"+order.SN+"/wifi")
                            }
                          }}
                        >
                          <SvgIcon 
                            fontSize="small" 
                            sx={{cursor: 'pointer'}} 
                          >
                            <ArrowTopRightOnSquareIcon />
                          </SvgIcon>
                        </Button>
                      </Tooltip>}
                      <Tooltip title={t('overview.latestOrders.editAlias')}>
                        <Button
                          onClick={()=>{
                            setDeviceToBeChanged(index)
                            setDeviceAlias(order.Alias)
                            setShowSetDeviceAlias(true)
                          }}
                        >
                          <SvgIcon 
                            fontSize="small" 
                            sx={{cursor: 'pointer'}} 
                          >
                            <PencilIcon />
                          </SvgIcon>
                        </Button>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </Box>
      </Scrollbar>
      {/*<Divider />
        <CardActions sx={{ justifyContent: 'flex-end' }}>
          <Button
            color="inherit"
            endIcon={(
              <SvgIcon fontSize="small">
                <ArrowRightIcon />
              </SvgIcon>
            )}
            size="small"
            variant="text"
          >
            View all
          </Button>
            </CardActions>*/}
    </Card>
    {showSetDeviceAlias&&
    <Dialog open={showSetDeviceAlias}>
      <DialogContent>
        <InputLabel>{t('overview.latestOrders.deviceAlias')}</InputLabel>
        <Input value={deviceAlias} onChange={(e)=>{setDeviceAlias(e.target.value)}}                          
        onKeyUp={e => {
          if (e.key === 'Enter') {
            setNewDeviceAlias(deviceAlias, orders[deviceToBeChanged].SN)
          }
        }}>
        </Input>
      </DialogContent>
      <DialogActions>
        <Button onClick={()=>{
          setShowSetDeviceAlias(false)
          setDeviceAlias(null)
          setDeviceToBeChanged(null)
        }}>{t('overview.latestOrders.cancel')}</Button>
        <Button onClick={()=>{
          setNewDeviceAlias(deviceAlias, orders[deviceToBeChanged].SN)
        }}>{t('overview.latestOrders.save')}</Button>
      </DialogActions>
    </Dialog>}
    </div>
  );
};

OverviewLatestOrders.prototype = {
  orders: PropTypes.array,
  sx: PropTypes.object
};
