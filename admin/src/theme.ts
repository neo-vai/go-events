import { defaultTheme } from 'react-admin';
import { createTheme } from '@mui/material/styles';

const muiTheme = createTheme({
    palette: {
        mode: 'light',
        primary: {
            main: '#1976d2',
        },
        secondary: {
            main: '#dc004e',
        },
        background: {
            default: '#f9fafb',
        },
    },
    typography: {
        fontFamily: '"Inter", "Roboto", "Helvetica", "Arial", sans-serif',
        h5: { fontWeight: 600 },
        h6: { fontWeight: 600 },
    },
    shape: { borderRadius: 12 },
    components: {
        MuiCard: {
            styleOverrides: {
                root: {
                    borderRadius: 16,
                    boxShadow: '0 4px 12px rgba(0,0,0,0.05)',
                },
            },
        },
        MuiPaper: {
            styleOverrides: {
                root: {
                    borderRadius: 16,
                    boxShadow: '0 4px 12px rgba(0,0,0,0.05)',
                },
            },
        },
        MuiButton: {
            styleOverrides: {
                root: {
                    borderRadius: 8,
                    textTransform: 'none',
                    fontWeight: 500,
                },
            },
        },
        MuiAppBar: {
            styleOverrides: {
                root: {
                    boxShadow: '0 1px 3px rgba(0,0,0,0.05)',
                },
            },
        },
        MuiDrawer: {
            styleOverrides: {
                paper: {
                    borderRight: 'none',
                    boxShadow: '2px 0 8px rgba(0,0,0,0.02)',
                },
            },
        },
    },
});

export const lightTheme = {
    ...defaultTheme,
    ...muiTheme,
    components: {
        ...defaultTheme.components,
        ...muiTheme.components,
        RaMenuItemLink: {
            styleOverrides: {
                root: {
                    borderRadius: 8,
                    margin: '4px 8px',
                    '&.RaMenuItemLink-active': {
                        backgroundColor: 'rgba(25, 118, 210, 0.08)',
                        borderLeft: '4px solid #1976d2',
                    },
                },
            },
        },
        RaSidebar: {
            styleOverrides: {
                root: {
                    backgroundColor: '#ffffff',
                },
            },
        },
        RaLayout: {
            styleOverrides: {
                root: {
                    '& .RaLayout-content': {
                        padding: '24px',
                    },
                },
            },
        },
    },
};