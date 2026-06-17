import { common } from '@mui/material/colors';
import { alpha } from '@mui/material/styles';
import { error, indigo, info, neutral, success, warning, graphics } from './colors';

const getColorScheme = () => {
  return JSON.stringify({
    "buttons": process.env.NEXT_PUBLIC_COLOR_PRIMARY || "#2563EB",
    "sidebar_end": process.env.NEXT_PUBLIC_COLOR_SIDEBAR_END || "#1D4ED8",
    "sidebar_initial": process.env.NEXT_PUBLIC_COLOR_SIDEBAR_START || "#0D3D3D",
    "tables": process.env.NEXT_PUBLIC_COLOR_TABLE_HEADER || "#1E3A5F",
    "words_outside_sidebar": process.env.NEXT_PUBLIC_COLOR_TEXT || "#1E293B",
    "connected_mtps_color": process.env.NEXT_PUBLIC_COLOR_ACCENT || "#0EA5A4"
  });
}

export function createPalette() {

  let colors = getColorScheme();
  console.log("colors scheme:", colors);

  let neutralColors = neutral(colors);

  return { 
    action: {
      active: neutralColors[500],
      disabled: alpha(neutralColors[900], 0.38),
      disabledBackground: alpha(neutralColors[900], 0.12),
      focus: alpha(neutralColors[900], 0.16),
      hover: alpha(neutralColors[900], 0.04),
      selected: alpha(neutralColors[900], 0.12)
    },
    background: {
      default: common.white,
      paper: common.white
    },
    divider: '#F2F4F7',
    error,
    graphics,
    info,
    mode: 'light',
    neutral: neutralColors,
    primary: indigo(colors),
    success,
    text: {
      primary: neutralColors[900],
      secondary: neutralColors[500],
      disabled: alpha(neutralColors[900], 0.38)
    },
    warning
  };
}
