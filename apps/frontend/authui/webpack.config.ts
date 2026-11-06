import { withFederation } from '../../../tools/module-federation/webpack';
import config from './module-federation.config';

export default withFederation(config);
