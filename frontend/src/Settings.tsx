import type { Component } from "solid-js";
import { createSignal } from "solid-js";

const Settings: Component = () => {
  const [address, setAddress] = createSignal("");
  const [name, setName] = createSignal("");
  const [password, setPassword] = createSignal("");
  const [commandPassword, setCommandPassword] = createSignal("");

  const setWithCurrentValue = (setFunc: Function) => (e: Event) => setFunc((e.currentTarget as HTMLInputElement).value)

  return (
    <div>
      <header class='container mx-auto'>
        <form>
          <div class="my-2">
            <label for="address" class="label-text">Address:</label>
            <input class="input w-full max-w-xs" type="text" name='address' value={address()}
              onchange={setWithCurrentValue(setAddress)} />
          </div>
          <div>
            <label for="name" class="label-text">Name:</label>
            <input type="text" name='name' class="input w-full max-w-xs" value={name()}
              onchange={setWithCurrentValue(setName)} />
          </div>
          <div>
            <label for="password" class="label-text">Password:</label>
            <input type="text" name='password' class="input w-full max-w-xs" value={password()}
              onchange={setWithCurrentValue(setPassword)} />
          </div>
          <div>
            <label for="commandPassword" class="label-text">Command Password:</label>
            <input type="text" name='commandPassword' class="input w-full max-w-xs" value={commandPassword()}
              onchange={setWithCurrentValue(setCommandPassword)} />
          </div>
          <div>
            <button class='btn btn-primary' type='submit'>Submit</button>
          </div>
        </form>
      </header>
    </div>
  );
}

export default Settings;
