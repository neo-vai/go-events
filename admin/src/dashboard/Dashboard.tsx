import { useState, useEffect } from 'react';
import { Card, CardContent, Typography, Grid } from '@mui/material';
import PeopleIcon from '@mui/icons-material/People';
import EventIcon from '@mui/icons-material/Event';
import VpnKeyIcon from '@mui/icons-material/VpnKey';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';

interface Stats {
    totalAccounts: number;
    activeAccounts: number;
    totalEvents: number;
    eventsToday: number;
    totalApiKeys: number;
    activeApiKeys: number;
}

const StatCard = ({ title, value, icon }: { title: string; value: number; icon: React.ReactNode }) => (
    <Card sx={{ height: '100%' }}>
        <CardContent sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div>
                <Typography color="textSecondary" gutterBottom variant="subtitle2">
                    {title}
                </Typography>
                <Typography variant="h4">{value}</Typography>
            </div>
            <div style={{ color: '#1976d2' }}>{icon}</div>
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
        return <Typography>Loading...</Typography>;
    }

    return (
        <Grid container spacing={3} sx={{ p: 3 }}>
            <Grid item xs={12} sm={6} md={3}>
                <StatCard title="Total Accounts" value={statsData.totalAccounts} icon={<PeopleIcon fontSize="large" />} />
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
                <StatCard title="Active Accounts" value={statsData.activeAccounts} icon={<CheckCircleIcon fontSize="large" />} />
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
                <StatCard title="Total Events" value={statsData.totalEvents} icon={<EventIcon fontSize="large" />} />
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
                <StatCard title="Events Today" value={statsData.eventsToday} icon={<EventIcon fontSize="large" />} />
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
                <StatCard title="Total API Keys" value={statsData.totalApiKeys} icon={<VpnKeyIcon fontSize="large" />} />
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
                <StatCard title="Active API Keys" value={statsData.activeApiKeys} icon={<CheckCircleIcon fontSize="large" />} />
            </Grid>
        </Grid>
    );
};

export default Dashboard;