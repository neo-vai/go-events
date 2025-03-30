import {
    List,
    Datagrid,
    TextField,
    DateField,
    Show,
    SimpleShowLayout,
    TextInput,
    ReferenceField,
    ReferenceInput,
} from 'react-admin';

const eventFilters = [
    <TextInput source="q" label="Search" alwaysOn />,
    <ReferenceInput source="account_id" reference="accounts" />,
    <TextInput source="api_key_id" label="API Key ID" />,
];

export const EventList = () => (
    <List filters={eventFilters}>
        <Datagrid rowClick="show">
            <TextField source="id" />
            <ReferenceField source="accountId" reference="accounts" link="show" />
            <TextField source="username" />
            <TextField source="name" />
            <TextField source="apiKeyId" />
            <DateField source="createdAt" showTime />
        </Datagrid>
    </List>
);

export const EventShow = () => (
    <Show>
        <SimpleShowLayout>
            <TextField source="id" />
            <ReferenceField source="accountId" reference="accounts" link="show" />
            <TextField source="username" />
            <TextField source="name" />
            <TextField source="apiKeyId" />
            <TextField source="payload" />
            <DateField source="createdAt" showTime />
        </SimpleShowLayout>
    </Show>
);