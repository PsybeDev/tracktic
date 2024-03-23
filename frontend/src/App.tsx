import type { Component } from 'solid-js';
import Menu from './Menu';

const App: Component = (props) => {
  return (
    <div class="grid grid-cols-4 gap-4">
      <div class="min-h-screen">
        <Menu />
      </div>
      <div class="col-span-3 min-h-screen">
        <section class="container mx-auto p-5">
          {props.children}
        </section>
      </div>
    </div>
  )
};

export default App;
