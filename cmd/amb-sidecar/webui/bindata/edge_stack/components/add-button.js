const template = document.createElement('template');
template.innerHTML = `
  <style>
    .container {
      padding: 8px;
    }
    button {
      display: block;
      float: right;
      overflow: hidden;
      position: relative;
      padding: 0 0 3px 1px;
      font-size: 16px;
      font-weight: bold;
      text-overflow: ellipsis;
      white-space: nowrap;
      cursor: pointer;
      outline: none;
      width: 1.7em;
      height: 1.7em;
      box-sizing: border-box;
      border: 1px solid #a1a1a1;
      border-radius: 50%;
      background: #ffffff;
      box-shadow: 0 2px 4px 0 rgba(0,0,0, 0.05), 0 2px 8px 0 rgba(161,161,161, 0.4);
      color: #363636;
    }
    button:focus {
      background-color: red;
    }
    :host {
        position: absolute;
        top: 0.1em;
        right: 0.1em;
    }
  </style>
  <div class="container" tabindex="1">
    <button>+++</button>
  </div>
`;
class AddButton extends HTMLElement {
  constructor() {
    super();
    this._shadowRoot = this.attachShadow({ mode: 'open' });
    this._shadowRoot.appendChild(template.content.cloneNode(true));
  }
}
window.customElements.define('add-button', AddButton);
