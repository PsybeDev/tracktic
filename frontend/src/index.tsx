/* @refresh reload */
import { render } from 'solid-js/web';
import { Router, Route } from '@solidjs/router';
import Settings from './Settings';
import "./tailwind.css";

import App from './App';

const Home = () => (
  <h1>Home</h1>
)

render(
  () => (
    <Router root={App}>
      <Route path="/" component={Home} />
      <Route path="/settings" component={Settings} />
    </Router>
    ),
  document.getElementById('root') as HTMLElement);
