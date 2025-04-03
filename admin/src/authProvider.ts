import { AuthProvider } from 'react-admin';

const authProvider: AuthProvider = {
    login: async ({ username, password }) => {
        const response = await fetch('/api/v1/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email: username, password }),
        });
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Invalid credentials');
        }
        const data = await response.json();
        localStorage.setItem('token', data.token);
        localStorage.setItem('accountId', data.accountId);
        const meResp = await fetch('/api/v1/account', {
            headers: { Authorization: `Bearer ${data.token}` },
        });
        if (!meResp.ok) {
            throw new Error('Failed to fetch account details');
        }
        const account = await meResp.json();
        if (account.role !== 'admin') {
            throw new Error('Access denied: admin role required');
        }
        localStorage.setItem('role', account.role);
        return Promise.resolve();
    },
    logout: () => {
        localStorage.removeItem('token');
        localStorage.removeItem('accountId');
        localStorage.removeItem('role');
        return Promise.resolve();
    },
    checkAuth: async () => {
        const token = localStorage.getItem('token');
        if (!token) {
            return Promise.reject();
        }
        try {
            const response = await fetch('/api/v1/account', {
                headers: { Authorization: `Bearer ${token}` },
            });
            if (!response.ok) {
                throw new Error('Invalid token');
            }
            const account = await response.json();
            if (account.role !== 'admin') {
                throw new Error('Not admin');
            }
            return Promise.resolve();
        } catch (error) {
            localStorage.removeItem('token');
            localStorage.removeItem('accountId');
            localStorage.removeItem('role');
            return Promise.reject();
        }
    },
    checkError: (error) => {
        const status = error.status;
        if (status === 401 || status === 403) {
            localStorage.removeItem('token');
            localStorage.removeItem('accountId');
            localStorage.removeItem('role');
            return Promise.reject();
        }
        return Promise.resolve();
    },
    getPermissions: () => Promise.resolve(),
    getIdentity: async () => {
        const token = localStorage.getItem('token');
        if (!token) return Promise.reject();
        const response = await fetch('/api/v1/account', {
            headers: { Authorization: `Bearer ${token}` },
        });
        if (!response.ok) {
            throw new Error('Failed to fetch identity');
        }
        const account = await response.json();
        return {
            id: account.id,
            fullName: account.email,
            avatar: undefined,
        };
    },
};

export default authProvider;