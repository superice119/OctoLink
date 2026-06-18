import { useCallback, useState } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  IconButton,
  Stack,
  SvgIcon,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import BoltIcon from '@heroicons/react/24/solid/BoltIcon';
import ArrowPathIcon from '@heroicons/react/24/outline/ArrowPathIcon';
import PencilSquareIcon from '@heroicons/react/24/outline/PencilSquareIcon';
import MagnifyingGlassIcon from '@heroicons/react/24/solid/MagnifyingGlassIcon';
import { useRouter } from 'next/router';
import { useBackendContext } from 'src/contexts/backend-context';

const DEFAULT_PATHS = 'Device.DeviceInfo.\nDevice.LANConfigSecurity.\nDevice.LocalAgent.';

export const DevicesParamQuery = () => {
  const router = useRouter();
  const { httpRequest } = useBackendContext();
  const deviceID = router.query.id?.[0];

  const [pathInput, setPathInput] = useState(DEFAULT_PATHS);
  const [maxDepth, setMaxDepth] = useState(0);
  const [results, setResults] = useState(null);
  const [cacheHit, setCacheHit] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  // SET dialog state
  const [editDialog, setEditDialog] = useState({ open: false, path: '', value: '' });
  const [setLoading2, setSetLoading] = useState(false);

  const parsePaths = () =>
    pathInput
      .split('\n')
      .map((p) => p.trim())
      .filter(Boolean);

  const queryParams = useCallback(
    async (useCached) => {
      const paths = parsePaths();
      if (!paths.length) return;

      setLoading(true);
      setError(null);
      setCacheHit(false);

      const endpoint = useCached
        ? `/api/device/${deviceID}/any/get/cached`
        : `/api/device/${deviceID}/any/get`;

      const { result, status, headers } = await httpRequest(
        endpoint,
        'PUT',
        JSON.stringify({ param_paths: paths, max_depth: maxDepth }),
        null,
      );

      setLoading(false);

      if (status !== 200) {
        setError(JSON.stringify(result, null, 2));
        return;
      }

      // X-Cache header comes through on cache hits
      if (headers && headers.get && headers.get('x-cache') === 'HIT') {
        setCacheHit(true);
      }

      // Flatten req_path_results → resolved_path_results → result_params
      const flat = [];
      (result?.req_path_results || []).forEach((rpr) => {
        (rpr.resolved_path_results || []).forEach((resolved) => {
          const objPath = resolved.resolved_path || '';
          Object.entries(resolved.result_params || {}).forEach(([paramName, value]) => {
            flat.push({ path: objPath + paramName, value });
          });
        });
      });
      setResults(flat);
    },
    [deviceID, httpRequest, pathInput, maxDepth],
  );

  const handleSet = async () => {
    setSetLoading(true);
    const body = {
      allow_partial: false,
      update_objs: [
        {
          obj_path: editDialog.path.replace(/[^.]+$/, ''),
          param_settings: [
            {
              param: editDialog.path.split('.').pop(),
              value: editDialog.value,
              required: true,
            },
          ],
        },
      ],
    };

    const { status } = await httpRequest(
      `/api/device/${deviceID}/any/set`,
      'PUT',
      JSON.stringify(body),
      null,
    );
    setSetLoading(false);
    setEditDialog({ open: false, path: '', value: '' });

    if (status === 200) {
      // Re-query live to show fresh value
      queryParams(false);
    }
  };

  return (
    <Card>
      <CardHeader
        avatar={<SvgIcon><MagnifyingGlassIcon /></SvgIcon>}
        title="Parameter Query"
        subheader="Enter TR-181 paths (one per line). Use cached endpoint to avoid round-trips; Refresh Live for guaranteed fresh data."
      />
      <Divider />
      <CardContent>
        <Stack spacing={2}>
          <Stack direction="row" spacing={2} alignItems="flex-start">
            <TextField
              label="TR-181 Parameter Paths"
              multiline
              rows={5}
              fullWidth
              value={pathInput}
              onChange={(e) => setPathInput(e.target.value)}
              placeholder={DEFAULT_PATHS}
              variant="outlined"
            />
            <Stack spacing={1} minWidth={130}>
              <TextField
                label="Max Depth"
                type="number"
                size="small"
                value={maxDepth}
                onChange={(e) => setMaxDepth(Number(e.target.value))}
                inputProps={{ min: 0, max: 10 }}
              />
              <Button
                variant="contained"
                startIcon={<SvgIcon><BoltIcon /></SvgIcon>}
                onClick={() => queryParams(true)}
                disabled={loading}
                fullWidth
              >
                Query (Cached)
              </Button>
              <Button
                variant="outlined"
                startIcon={<SvgIcon><ArrowPathIcon /></SvgIcon>}
                onClick={() => queryParams(false)}
                disabled={loading}
                fullWidth
              >
                Refresh Live
              </Button>
            </Stack>
          </Stack>

          {loading && (
            <Box display="flex" justifyContent="center" py={3}>
              <CircularProgress />
            </Box>
          )}

          {error && (
            <Box>
              <Typography color="error" variant="body2" fontWeight="bold">Error:</Typography>
              <pre style={{ fontSize: 12, color: '#c00', overflow: 'auto' }}>{error}</pre>
            </Box>
          )}

          {results !== null && !loading && (
            <Box>
              <Stack direction="row" spacing={1} alignItems="center" mb={1}>
                <Typography variant="subtitle2">
                  {results.length} parameter{results.length !== 1 ? 's' : ''} retrieved
                </Typography>
                {cacheHit && (
                  <Chip label="From Cache (5 min TTL)" size="small" color="info" />
                )}
              </Stack>
              <Box sx={{ overflowX: 'auto' }}>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell><strong>Parameter Path</strong></TableCell>
                      <TableCell><strong>Value</strong></TableCell>
                      <TableCell align="right"><strong>Action</strong></TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {results.map(({ path, value }) => (
                      <TableRow key={path} hover>
                        <TableCell sx={{ fontFamily: 'monospace', fontSize: 12 }}>{path}</TableCell>
                        <TableCell sx={{ fontFamily: 'monospace', fontSize: 12, maxWidth: 300, wordBreak: 'break-all' }}>
                          {value}
                        </TableCell>
                        <TableCell align="right">
                          <Tooltip title="Set value">
                            <IconButton
                              size="small"
                              onClick={() => setEditDialog({ open: true, path, value })}
                            >
                              <SvgIcon fontSize="small"><PencilSquareIcon /></SvgIcon>
                            </IconButton>
                          </Tooltip>
                        </TableCell>
                      </TableRow>
                    ))}
                    {results.length === 0 && (
                      <TableRow>
                        <TableCell colSpan={3} align="center">
                          <Typography variant="body2" color="text.secondary">
                            No parameters returned for these paths.
                          </Typography>
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </Box>
            </Box>
          )}
        </Stack>
      </CardContent>

      {/* SET dialog */}
      <Dialog open={editDialog.open} maxWidth="sm" fullWidth>
        <DialogTitle>Set Parameter</DialogTitle>
        <DialogContent>
          <Stack spacing={2} pt={1}>
            <TextField
              label="Path"
              value={editDialog.path}
              disabled
              fullWidth
              size="small"
              inputProps={{ style: { fontFamily: 'monospace' } }}
            />
            <TextField
              label="New Value"
              value={editDialog.value}
              onChange={(e) => setEditDialog((d) => ({ ...d, value: e.target.value }))}
              fullWidth
              autoFocus
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditDialog({ open: false, path: '', value: '' })}>
            Cancel
          </Button>
          {setLoading2 ? (
            <CircularProgress size={24} />
          ) : (
            <Button variant="contained" onClick={handleSet}>
              Apply
            </Button>
          )}
        </DialogActions>
      </Dialog>
    </Card>
  );
};
