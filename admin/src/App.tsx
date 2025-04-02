import { Admin, Resource, Login } from 'react-admin';
import { lightTheme } from './theme';
import authProvider from './authProvider';
import { dataProvider } from './dataProvider';
import { AccountList, AccountEdit, AccountCreate } from './resources/accounts';
import { EventList, EventShow } from './resources/events';
import { ApiKeyList, ApiKeyEdit } from './resources/apiKeys';
import Dashboard from './dashboard/Dashboard';
import PeopleIcon from '@mui/icons-material/People';
import EventIcon from '@mui/icons-material/Event';
import VpnKeyIcon from '@mui/icons-material/VpnKey';

const App = () => (
    <Admin
        theme={lightTheme}
        authProvider={authProvider}
        dataProvider={dataProvider}
        dashboard={Dashboard}
        loginPage={Login}
        requireAuth
    >
        <Resource
            name="accounts"
            list={AccountList}
            edit={AccountEdit}
            create={AccountCreate}
            icon={PeopleIcon}
        />
        <Resource
            name="events"
            list={EventList}
            show={EventShow}
            icon={EventIcon}
        />
        <Resource
            name="api-keys"
            list={ApiKeyList}
            edit={ApiKeyEdit}
            icon={VpnKeyIcon}
            options={{ label: 'API Keys' }}
        />
    </Admin>
);

export default App;