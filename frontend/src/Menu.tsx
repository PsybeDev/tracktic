import type { Component } from "solid-js";
import { A } from '@solidjs/router';

const Menu: Component = () => {
  return (
  <ul class="menu bg-base-200 w-56 rounded-box">
      <li><A href="/" end>Home</A></li>
      <li><A href="/settings">Settings</A></li>
  </ul>
  )
}

export default Menu;
