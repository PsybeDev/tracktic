import type { Component } from "solid-js";
import { createSignal } from "solid-js";

const Settings: Component = () => {
  const [address, setAddress] = createSignal("");
  const [name, setName] = createSignal("");
  const [password, setPassword] = createSignal("");
  const [commandPassword, setCommandPassword] = createSignal("");

  const setWithCurrentValue = (setFunc: Function) => (e: Event) => setFunc((e.currentTarget as HTMLInputElement).value)

  return (
    <div class="container">
      <header>
        <h1>Settings</h1>
      </header>
      <div>
        <form>
          <div class="grid grid-cols-1 gap-4">
          <div class="my-2">
            <label class="input input-borderd w-full max-w-xs">
              IP 
              <input type="text" name='ip' value={address()} onchange={setWithCurrentValue(setAddress)} />
            </label>
          </div>
          <div class="my-2">
            <label class="input input-bordered w-full max-w-xs">
              Port 
              <input type="text" name='port' value={name()} onchange={setWithCurrentValue(setName)} />
            </label>
          </div>
          <div class="my-2">
            <label class="input input-bordered w-full max-w-xs">
              Password 
            <input type="text" name='password' placeholder="Password" value={password()}
              onchange={setWithCurrentValue(setPassword)} />
              </label>
          </div>
          <div class="my-2">
            <label class="input input-bordered w-full max-w-xs">
            Command Password 
            <input type="text" name='commandPassword' laceholder="Command Password" value={commandPassword()}
              onchange={setWithCurrentValue(setCommandPassword)} />
              </label>
          </div>
          <div class="my-2">
            <button class='btn btn-primary' type='submit'>Submit</button>
          </div>
          </div>
        </form>
      </div>
    </div>
  );
}

export default Settings;
