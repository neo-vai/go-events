import { fetchUtils, DataProvider } from 'react-admin';

const apiUrl = '/api/v1/admin';

const httpClient = (url: string, options: fetchUtils.Options = {}) => {
    const token = localStorage.getItem('token');
    const headers = new Headers(options.headers);
    if (token) {
        headers.set('Authorization', `Bearer ${token}`);
    }
    return fetchUtils.fetchJson(url, { ...options, headers });
};

export const dataProvider: DataProvider = {
    getList: async (resource, params) => {
        const pagination = params.pagination || { page: 1, perPage: 20 };
        const sort = params.sort || { field: 'id', order: 'ASC' };
        const filter = params.filter || {};

        const { page, perPage } = pagination;
        const { field, order } = sort;

        const queryParams = new URLSearchParams();
        queryParams.append('_start', String((page - 1) * perPage));
        queryParams.append('_end', String(page * perPage));
        queryParams.append('_sort', field);
        queryParams.append('_order', order);
        if (Object.keys(filter).length > 0) {
            queryParams.append('filter', JSON.stringify(filter));
        }

        const url = `${apiUrl}/${resource}?${queryParams.toString()}`;
        const { json, headers } = await httpClient(url);
        const total = headers.get('X-Total-Count')
            ? parseInt(headers.get('X-Total-Count')!, 10)
            : json.length;
        return { data: json, total };
    },

    getOne: async (resource, params) => {
        const url = `${apiUrl}/${resource}/${params.id}`;
        const { json } = await httpClient(url);
        return { data: json };
    },

    getMany: async (resource, params) => {
        const query = {
            filter: JSON.stringify({ id: params.ids }),
        };
        const url = `${apiUrl}/${resource}?${new URLSearchParams(query).toString()}`;
        const { json } = await httpClient(url);
        return { data: json };
    },

    getManyReference: async (resource, params) => {
        const pagination = params.pagination || { page: 1, perPage: 20 };
        const sort = params.sort || { field: 'id', order: 'ASC' };
        const filter = params.filter || {};

        const { page, perPage } = pagination;
        const { field, order } = sort;

        const queryParams = new URLSearchParams();
        queryParams.append('_start', String((page - 1) * perPage));
        queryParams.append('_end', String(page * perPage));
        queryParams.append('_sort', field);
        queryParams.append('_order', order);

        const mergedFilter = {
            ...filter,
            [params.target]: params.id,
        };
        queryParams.append('filter', JSON.stringify(mergedFilter));

        const url = `${apiUrl}/${resource}?${queryParams.toString()}`;
        const { json, headers } = await httpClient(url);
        const total = headers.get('X-Total-Count')
            ? parseInt(headers.get('X-Total-Count')!, 10)
            : json.length;
        return { data: json, total };
    },

    create: async (resource, params) => {
        const url = `${apiUrl}/${resource}`;
        const { json } = await httpClient(url, {
            method: 'POST',
            body: JSON.stringify(params.data),
        });
        return { data: json };
    },

    update: async (resource, params) => {
        const url = `${apiUrl}/${resource}/${params.id}`;
        const { json } = await httpClient(url, {
            method: 'PUT',
            body: JSON.stringify(params.data),
        });
        return { data: json };
    },

    updateMany: async (resource, params) => {
        const url = `${apiUrl}/${resource}`;
        const { json } = await httpClient(url, {
            method: 'PUT',
            body: JSON.stringify({ ids: params.ids, data: params.data }),
        });
        return { data: json };
    },

    delete: async (resource, params) => {
        const url = `${apiUrl}/${resource}/${params.id}`;
        const { json } = await httpClient(url, {
            method: 'DELETE',
        });
        return { data: json };
    },

    deleteMany: async (resource, params) => {
        const url = `${apiUrl}/${resource}`;
        const { json } = await httpClient(url, {
            method: 'DELETE',
            body: JSON.stringify({ ids: params.ids }),
        });
        return { data: json };
    },
};