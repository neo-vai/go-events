// admin/src/dashboard/Dashboard.tsx
import { useState, useEffect } from 'react';
import { Card, CardContent, Typography, Grid, Box, Skeleton } from '@mui/material';
import PeopleIcon from '@mui/icons-material/People';
import EventIcon from '@mui/icons-material/Event';
import VpnKeyIcon from '@mui/icons-material/VpnKey';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';

interface Stats {
    totalAccounts: number;
    activeAccounts: number;
    totalEvents: number;
    eventsToday: number;
    totalApiKeys: number;
    activeApiKeys: number;
}

const StatCard = ({ title, value, icon, color }: { title: string; value: number; icon: React.ReactNode; color: string }) => (
    <Card sx={{ height: '100%', transition: 'transform 0.2s', '&:hover': { transform: 'translateY(-4px)' } }}>
        <CardContent>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
                <Box
                    sx={{
                        backgroundColor: color,
                        borderRadius: '12px',
                        p: 1,
                        display: 'inline-flex',
                        color: 'white',
                    }}
                >
                    {icon}
                </Box>
                <Typography variant="h4" component="div" fontWeight={700}>
                    {value}
                </Typography>
            </Box>
            <Typography color="textSecondary" variant="body2">
                {title}
            </Typography>
        </CardContent>
    </Card>
);

const Dashboard = () => {
    const [statsData, setStatsData] = useState<Stats | null>(null);
    const token = localStorage.getItem('token');

    useEffect(() => {
        fetch('/api/v1/admin/stats', {
            headers: { Authorization: `Bearer ${token}` },
        })
            .then((res) => {
                if (!res.ok) throw new Error('Failed to fetch stats');
                return res.json();
            })
            .then(setStatsData)
            .catch(console.error);
    }, [token]);

    if (!statsData) {
        return (
            <Grid container spacing={3} sx={{ p: 3 }}>
                {[...Array(6)].map((_, i) => (
                    <Grid item xs={12} sm={6} md={3} key={i}>
                        <Skeleton variant="rounded" height={120} />
                    </Grid>
                ))}
            </Grid>
        );
    }

    const cards = [
        { title: 'Total Accounts', value: statsData.totalAccounts, icon: <PeopleIcon />, color: '#6366f1' },
        { title: 'Active Accounts', value: statsData.activeAccounts, icon: <CheckCircleIcon />, color: '#10b981' },
        { title: 'Total Events', value: statsData.totalEvents, icon: <EventIcon />, color: '#f59e0b' },
        { title: 'Events Today', value: statsData.eventsToday, icon: <TrendingUpIcon />, color: '#3b82f6' },
        { title: 'Total API Keys', value: statsData.totalApiKeys, icon: <VpnKeyIcon />, color: '#8b5cf6' },
        { title: 'Active API Keys', value: statsData.activeApiKeys, icon: <CheckCircleIcon />, color: '#ec489a' },
    ];

    return (
        <Grid container spacing={3} sx={{ p: 3 }}>
            {cards.map((card, idx) => (
                <Grid item xs={12} sm={6} md={3} key={idx}>
                    <StatCard {...card} />
                </Grid>
            ))}
        </Grid>
    );
};

export default Dashboard;