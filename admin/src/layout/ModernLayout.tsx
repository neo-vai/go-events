import { Layout, LayoutProps } from 'react-admin';
import { ModernAppBar } from './ModernAppBar';

export const ModernLayout = (props: LayoutProps) => (
    <Layout {...props} appBar={ModernAppBar} />
);