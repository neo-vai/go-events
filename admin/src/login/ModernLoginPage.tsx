import { useState } from 'react';
import { useLogin, useNotify, useSafeSetState } from 'react-admin';
import {
    Card,
    CardContent,
    TextField,
    Button,
    Typography,
    Box,
    Paper,
    CircularProgress,
} from '@mui/material';

const ModernLoginPage = () => {
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [loading, setLoading] = useSafeSetState(false);
    const login = useLogin();
    const notify = useNotify();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setLoading(true);
        try {
            await login({ username: email, password });
        } catch (error: any) {
            notify(error.message || 'Invalid credentials', { type: 'error' });
            setLoading(false);
        }
    };

    return (
        <Box
            sx={{
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
                minHeight: '100vh',
                bgcolor: 'background.default',
            }}
        >
            <Paper elevation={3} sx={{ maxWidth: 400, width: '100%', mx: 2 }}>
                <Card>
                    <CardContent sx={{ p: 4 }}>
                        <Typography variant="h5" component="h1" textAlign="center" gutterBottom fontWeight={600}>
                            Event Tracker Admin
                        </Typography>
                        <Typography variant="body2" textAlign="center" color="textSecondary" sx={{ mb: 3 }}>
                            Sign in to continue
                        </Typography>
                        <form onSubmit={handleSubmit}>
                            <TextField
                                label="Email"
                                type="email"
                                variant="outlined"
                                fullWidth
                                margin="normal"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                required
                                autoFocus
                                sx={{ mb: 2 }}
                            />
                            <TextField
                                label="Password"
                                type="password"
                                variant="outlined"
                                fullWidth
                                margin="normal"
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                required
                                sx={{ mb: 3 }}
                            />
                            <Button
                                type="submit"
                                variant="contained"
                                fullWidth
                                size="large"
                                disabled={loading}
                                sx={{ py: 1.5 }}
                            >
                                {loading ? <CircularProgress size={24} /> : 'Login'}
                            </Button>
                        </form>
                    </CardContent>
                </Card>
            </Paper>
        </Box>
    );
};

export default ModernLoginPage;