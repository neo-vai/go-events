import { AuthProvider } from 'react-admin';

const authProvider: AuthProvider = {
    login: async ({ username, password }) => {
        const response = await fetch('/api/v1/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ login: username, password }),
        });
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Invalid credentials');
        }
        const data = await response.json();
        localStorage.setItem('token', data.token);
        localStorage.setItem('accountId', data.accountId);
        // After login, fetch current account to verify admin role
        const meResp = await fetch('/api/v1/account', {
            headers: { Authorization: `Bearer ${data.token}` },
        });
        if (!meResp.ok) {
            throw new Error('Failed to fetch account details');
        }
        // The backend should provide role in response, but current /account doesn't return role.
        // We trust that backend will reject non-admin on protected routes.
        return Promise.resolve();
    },
    logout: () => {
        localStorage.removeItem('token');
        localStorage.removeItem('accountId');
        return Promise.resolve();
    },
    checkAuth: () => {
        return localStorage.getItem('token') ? Promise.resolve() : Promise.reject();
    },
    checkError: (error) => {
        const status = error.status;
        if (status === 401 || status === 403) {
            localStorage.removeItem('token');
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
            fullName: account.name || account.login,
            avatar: undefined,
        };
    },
};

export default authProvider;