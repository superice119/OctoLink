import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import LanguageIcon from '@heroicons/react/24/outline/LanguageIcon';
import {
  IconButton,
  Menu,
  MenuItem,
  ListItemText,
  SvgIcon,
  Tooltip,
} from '@mui/material';
import { usePopover } from 'src/hooks/use-popover';
import { SUPPORTED_LANGUAGES } from 'src/i18n';

export const LanguageSwitcher = () => {
  const { t, i18n } = useTranslation();
  const popover = usePopover();

  const current = (i18n.resolvedLanguage || i18n.language || 'zh').split('-')[0];

  const handleSelect = useCallback(
    (lng) => {
      i18n.changeLanguage(lng);
      if (typeof document !== 'undefined') {
        document.documentElement.lang = lng;
      }
      popover.handleClose();
    },
    [i18n, popover]
  );

  return (
    <>
      <Tooltip title={t('language.label')}>
        <IconButton
          onClick={popover.handleOpen}
          ref={popover.anchorRef}
          aria-label={t('language.label')}
        >
          <SvgIcon fontSize="small">
            <LanguageIcon />
          </SvgIcon>
        </IconButton>
      </Tooltip>
      <Menu
        anchorEl={popover.anchorRef.current}
        open={popover.open}
        onClose={popover.handleClose}
        anchorOrigin={{ horizontal: 'right', vertical: 'bottom' }}
        transformOrigin={{ horizontal: 'right', vertical: 'top' }}
      >
        {SUPPORTED_LANGUAGES.map((lng) => (
          <MenuItem
            key={lng}
            selected={current === lng}
            onClick={() => handleSelect(lng)}
          >
            <ListItemText primary={t(`language.${lng}`)} />
          </MenuItem>
        ))}
      </Menu>
    </>
  );
};
