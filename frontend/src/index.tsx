/* @refresh reload */
import { render } from 'solid-js/web';
import { Router, Route } from '@solidjs/router';
import Settings from './Settings';
import LiveTiming from './views/LiveTiming'
import "./tailwind.css";

import App from './App';

render(
  () => (
    <Router root={App}>
      <Route path="/" component={LiveTiming} />
      <Route path="/settings" component={Settings} />
    </Router>
  ),
  document.getElementById('root') as HTMLElement);
