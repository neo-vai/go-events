import { AppBar, Toolbar, IconButton, Typography, Box } from '@mui/material';
import MenuIcon from '@mui/icons-material/Menu';
import RefreshIcon from '@mui/icons-material/Refresh';
import AccountCircleIcon from '@mui/icons-material/AccountCircle';
import { useSidebarState } from 'react-admin';

export const ModernAppBar = (props: any) => {
    const [open, setOpen] = useSidebarState();

    const toggleSidebar = () => {
        setOpen(!open);
    };

    return (
        <AppBar
            {...props}
            position="fixed"
            elevation={0}
            sx={{
                height: 64,
                backgroundColor: '#ffffff',
                borderBottom: '1px solid #e2e8f0',
                color: '#0f172a',
                zIndex: 1300,
            }}
        >
            <Toolbar disableGutters sx={{ px: 2, height: 64, minHeight: 64 }}>
                <IconButton
                    color="inherit"
                    onClick={toggleSidebar}
                    edge="start"
                    sx={{ mr: 2 }}
                >
                    <MenuIcon />
                </IconButton>
                <Typography variant="h6" sx={{ flexGrow: 1, fontWeight: 600 }}>
                    Event Tracker Admin
                </Typography>
                <Box sx={{ display: 'flex', gap: 1 }}>
                    <IconButton color="inherit" onClick={() => window.location.reload()}>
                        <RefreshIcon />
                    </IconButton>
                    <IconButton color="inherit">
                        <AccountCircleIcon />
                    </IconButton>
                </Box>
            </Toolbar>
        </AppBar>
    );
};