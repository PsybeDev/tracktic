import type { Component } from 'solid-js';
import Menu from './Menu';

const App: Component = (props) => {
  return (
    <div class="grid grid-cols-4 gap-4">
      <div>
        <Menu />
      </div>
      <div class="col-span-3">
        {props.children}
      </div>
    </div>
  )
};

export default App;
