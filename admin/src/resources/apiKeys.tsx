import {
    List,
    Datagrid,
    TextField,
    BooleanField,
    DateField,
    Edit,
    SimpleForm,
    BooleanInput,
    ReferenceField,
    ReferenceInput,
    TextInput,
} from 'react-admin';

const apiKeyFilters = [
    <TextInput source="q" label="Search" alwaysOn />,
    <ReferenceInput source="account_id" reference="accounts" />,
    <BooleanInput source="active" />,
];

export const ApiKeyList = () => (
    <List filters={apiKeyFilters}>
        <Datagrid rowClick="edit">
            <TextField source="id" />
            <ReferenceField source="accountId" reference="accounts" link="show" />
            <TextField source="keyPrefix" label="Key Prefix" />
            <BooleanField source="active" />
            <DateField source="createdAt" showTime />
        </Datagrid>
    </List>
);

export const ApiKeyEdit = () => (
    <Edit>
        <SimpleForm>
            <TextField source="id" />
            <ReferenceField source="accountId" reference="accounts" link="show" />
            <TextField source="keyPrefix" label="Key Prefix" />
            <BooleanInput source="active" />
            <DateField source="createdAt" showTime />
        </SimpleForm>
    </Edit>
);