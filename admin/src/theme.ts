// admin/src/theme.ts
import { createTheme } from '@mui/material/styles';

export const lightTheme = createTheme({
    palette: {
        mode: 'light',
        primary: {
            main: '#6366f1',
            light: '#818cf8',
            dark: '#4f46e5',
            contrastText: '#ffffff',
        },
        secondary: {
            main: '#10b981',
            light: '#34d399',
            dark: '#059669',
        },
        error: {
            main: '#f59e0b',
        },
        background: {
            default: '#f8fafc',
            paper: '#ffffff',
        },
        text: {
            primary: '#0f172a',
            secondary: '#475569',
        },
    },
    typography: {
        fontFamily: '"Inter", "Roboto", "Helvetica", "Arial", sans-serif',
        h5: { fontWeight: 600 },
        h6: { fontWeight: 600 },
    },
    shape: { borderRadius: 12 },
    components: {
        MuiAppBar: {
            styleOverrides: {
                root: {
                    backgroundColor: '#ffffff',
                    boxShadow: '0px 1px 3px rgba(0,0,0,0.05)',
                    borderBottom: '1px solid #e2e8f0',
                    color: '#0f172a',
                },
            },
        },
        MuiCard: {
            styleOverrides: {
                root: {
                    borderRadius: 16,
                    boxShadow: '0px 1px 3px rgba(0,0,0,0.05)',
                    border: '1px solid #e2e8f0',
                },
            },
        },
        MuiButton: {
            styleOverrides: {
                root: {
                    borderRadius: 8,
                    textTransform: 'none',
                },
            },
        },
        RaMenuItemLink: {
            styleOverrides: {
                root: {
                    borderRadius: 8,
                    margin: '4px 8px',
                    '&.RaMenuItemLink-active': {
                        backgroundColor: 'rgba(99, 102, 241, 0.08)',
                        borderLeft: '3px solid #6366f1',
                    },
                },
            },
        },
    },
});