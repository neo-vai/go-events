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
            <TextField source="name" />
            <EmailField source="email" />
            <TextField source="login" />
            <TextField source="role" />
            <BooleanField source="active" />
            <DateField source="createdAt" showTime />
        </Datagrid>
    </List>
);

export const AccountEdit = () => (
    <Edit>
        <SimpleForm>
            <TextInput source="name" validate={required()} />
            <TextInput source="email" validate={[required(), email()]} />
            <TextInput source="login" validate={required()} />
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
            <TextInput source="name" validate={required()} />
            <TextInput source="email" validate={[required(), email()]} />
            <TextInput source="login" validate={required()} />
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