import { fetchUtils, DataProvider } from 'react-admin';
import simpleRestProvider from 'ra-data-simple-rest';

const httpClient = (url: string, options: fetchUtils.Options = {}) => {
    const token = localStorage.getItem('token');
    const headers = new Headers(options.headers);
    if (token) {
        headers.set('Authorization', `Bearer ${token}`);
    }
    return fetchUtils.fetchJson(url, { ...options, headers });
};

const baseDataProvider = simpleRestProvider('/api/v1/admin', httpClient);

export const dataProvider: DataProvider = {
    ...baseDataProvider,
}