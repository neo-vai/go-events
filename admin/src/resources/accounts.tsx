import {
    List,
    Datagrid,
    TextField,
    EmailField,
    BooleanField,
    DateField,
    Edit,
    SimpleForm,
    TextInput,
    BooleanInput,
    SelectInput,
    Create,
    PasswordInput,
    required,
    email,
} from 'react-admin';

const accountFilters = [
    <TextInput source="q" label="Search" alwaysOn />,
    <SelectInput
        source="role"
        choices={[
            { id: 'user', name: 'User' },
            { id: 'admin', name: 'Admin' },
        ]}
    />,
    <BooleanInput source="active" />,
];

export const AccountList = () => (
    <List filters={accountFilters}>
        <Datagrid rowClick="edit">
            <TextField source="id" />
            <EmailField source="email" />
            <TextField source="role" />
            <BooleanField source="active" />
            <DateField source="createdAt" showTime />
        </Datagrid>
    </List>
);

export const AccountEdit = () => (
    <Edit>
        <SimpleForm>
            <TextInput source="email" validate={[required(), email()]} />
            <SelectInput
                source="role"
                choices={[
                    { id: 'user', name: 'User' },
                    { id: 'admin', name: 'Admin' },
                ]}
                validate={required()}
            />
            <BooleanInput source="active" />
        </SimpleForm>
    </Edit>
);

export const AccountCreate = () => (
    <Create>
        <SimpleForm>
            <TextInput source="email" validate={[required(), email()]} />
            <PasswordInput source="password" validate={required()} />
            <SelectInput
                source="role"
                choices={[
                    { id: 'user', name: 'User' },
                    { id: 'admin', name: 'Admin' },
                ]}
                defaultValue="user"
                validate={required()}
            />
        </SimpleForm>
    </Create>
);