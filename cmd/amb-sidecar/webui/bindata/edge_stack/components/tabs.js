import { Model } from '../mvc/framework/model.js'
import { View, html, css } from '../mvc/framework/view.js'
import {useContext} from './context.js';
import {HASH} from './hash.js';

/**
 * Provides a small wrapper around named slots, to properly
 * render one of a series of "tabs".
 */
export class Tabs extends View {
  /**
   * A list of properties to track for "re-render".
   *
   * NOTE: this is not guaranteed to be a full list of properties
   *       but only a list of items that when changed will trigger
   *       a re-render (these will be debounced if a lot of updates
   *       happen at once).
   */
  /* styles() returns the styles for the tab elements. */	
  static get styles() {	
    return css`
      .col_left { /* old stuff */
        /*
        float: left;
        width: 190px;
        background-color: black;
        color: white;
        height: 90vh;
        font-size: 85%;
        font-weight: bold;
        */
      }
      
      .col_right { /* old stuff */
        /*
        margin-left: 190px;
        */
      }
      
      * {
        margin: 0;
        padding: 0;
        border: 0;
        position: relative;
        box-sizing: border-box
      }
      
      *, textarea {
        vertical-align: top
      }
      
      .col_wrapper {
        display: flex;
      }
      
      .col_left {
      }
      
      .col_left, .col_right {
        display: -webkit-flex;
        display: -ms-flexbox;
        display: flex
      }
      
      .col_left, .col_right {
        -webkit-flex-direction: column;
        -ms-flex-direction: column;
        flex-direction: column
      }
      
      .col_left {
        background: #2e3147;
        -webkit-flex: 0 0 250px;
        -ms-flex: 0 0 250px;
        flex: 0 0 250px
      }
      
      .col_left .logo {
        display: -webkit-flex;
        display: -ms-flexbox;
        display: flex;
        -webkit-justify-content: center;
        -ms-flex-pack: center;
        justify-content: center;
        border-bottom: 6px solid #2e3147;
      }
      
      .col_left .logo {
        -webkit-align-content: center;
        -ms-flex-line-pack: center;
        align-content: center
      }
      
      .col_left .logo {
        -webkit-flex: 0 0 80px;
        -ms-flex: 0 0 80px;
        flex: 0 0 80px;
        background: #5f3eff;
        padding: 20px
      }
      
      .col_left .logo img {
        width: 80%;
        max-width: 150px
      }
      
      .col_right {
        -webkit-flex: 3 0 auto;
        -ms-flex: 3 0 auto;
        flex: 3 0 auto;
        background: #f3f3f3
      }
      
      navigation a, navigation a .label, navigation a .label .icon {
        display: -webkit-flex;
        display: -ms-flexbox;
        display: flex;
        -webkit-justify-content: center;
        -ms-flex-pack: center;
        justify-content: center
      }
      
      navigation a, navigation a .label, navigation a .label .icon {
        -webkit-align-content: center;
        -ms-flex-line-pack: center;
        align-content: center
      }
      
      navigation {
        display: block;
        width: 100%
      }
      
      navigation a {
        padding: 0;
        text-decoration: none;
        height: 53px;
        transition: all .7s ease
      }
      
      navigation a .selected_stripe {
        -webkit-flex: 0 0 10px;
        -ms-flex: 0 0 10px;
        flex: 0 0 10px;
        background: #ff4329;
        min-height: 100%;
        opacity: 0
      }
      
      navigation a, navigation a .label, navigation a .label .icon {
        -webkit-align-items: center;
        -ms-flex-align: center;
        align-items: center
      }
      
      navigation a .label {
        -webkit-flex-direction: row;
        -ms-flex-direction: row;
        flex-direction: row;
        margin-left: 6%;
        -webkit-flex: 3 0 0;
        -ms-flex: 3 0 0px;
        flex: 3 0 0
      }
      
      navigation a .label .icon {
        height: 100%;
        -webkit-flex: 0 0 25px;
        -ms-flex: 0 0 25px;
        flex: 0 0 25px
      }

      navigation a .label .icon img, navigation a.selected .label .icon img {
        width: 25px;
        height: 25px;
        max-height: 35px
      }

      navigation a .label .name {
        -webkit-flex: 1 0 auto;
        -ms-flex: 1 0 auto;
        flex: 1 0 auto;
        color: #9a9a9a;
        padding-left: 20px;
        font-size: 1rem;
        transition: all .7s ease
      }
      
      navigation a:hover {
        background: #363a58;
        transition: all .8s ease
      }
      
      navigation a:hover .label .icon svg path, navigation a:hover .label .icon svg polygon, navigation a:hover .label .icon svg rect {
        fill: #53f7d2;
        transition: fill .7s ease
      }

      navigation a:hover .label .icon img path, navigation a:hover .label .icon img polygon, navigation a:hover .label .icon img rect {
        fill: #53f7d2;
        transition: fill .7s ease
      }
      
      navigation a:hover .label .name {
        color: #53f7d2;
        transition: all .7s ease
      }
      
      navigation a.selected {
        background: #5f3eff;
        transition: all 2.8s ease
      }
      
      navigation a.selected .selected_stripe {
        -webkit-flex: 0 0 10px;
        -ms-flex: 0 0 10px;
        flex: 0 0 10px;
        background: #ff4329;
        min-height: 100%;
        opacity: 1
      }
      
      .content a.button_large, navigation a.selected .label {
        -webkit-align-content: center;
        -ms-flex-line-pack: center;
        align-content: center;
        -webkit-align-items: center;
        -ms-flex-align: center;
        align-items: center
      }
      
      .content, navigation a.selected .label {
        display: -webkit-flex;
        display: -ms-flexbox;
        display: flex
      }
      
      navigation a.selected .label, navigation a.selected .label .icon {
        -webkit-justify-content: center;
        -ms-flex-pack: center;
        justify-content: center
      }
      
      navigation a.selected .label {
        margin-left: 6%;
        -webkit-flex-direction: row;
        -ms-flex-direction: row;
        flex-direction: row;
        -webkit-flex: 3 0 0;
        -ms-flex: 3 0 0px;
        flex: 3 0 0
      }
      
      navigation a.selected .label .icon {
        height: 100%;
        -webkit-flex: 0 0 25px;
        -ms-flex: 0 0 25px;
        flex: 0 0 25px;
        display: -webkit-flex;
        display: -ms-flexbox;
        display: flex;
        -webkit-align-items: center;
        -ms-flex-align: center;
        align-items: center;
        -webkit-align-content: center;
        -ms-flex-line-pack: center;
        align-content: center
      }
      
      navigation a.selected .label .icon svg path, navigation a.selected .label .icon svg polygon, navigation a.selected .label .icon svg rect {
        fill: #fff;
        transition: fill .7s ease
      }
      
      navigation a.selected .label .name {
        -webkit-flex: 1 0 auto;
        -ms-flex: 1 0 auto;
        flex: 1 0 auto;
        color: #fff;
        padding-left: 20px;
        font-size: 1rem;
        transition: all .7s ease
      }
      
      .content {
        width: 100%;
        -webkit-flex-direction: column;
        -ms-flex-direction: column;
        flex-direction: column;
        max-width: 900px;
        margin: 0 auto;
        padding: 30px
      }
      
      .content a.button_large {
        display: -webkit-flex;
        display: -ms-flexbox;
        display: flex;
        -webkit-justify-content: center;
        -ms-flex-pack: center;
        justify-content: center
      }
      .tabLogo {
        filter: invert(58%) sepia(20%) saturate(0%) hue-rotate(262deg) brightness(103%) contrast(87%);
        transition: all .7s ease;
      }
      navigation a:hover .tabLogo {
        filter: invert(95%) sepia(93%) saturate(687%) hue-rotate(85deg) brightness(101%) contrast(94%);
        transition: all .7s ease;
      }
      navigation a.selected img.tabLogo {
        filter: invert(99%) sepia(37%) saturate(0%) hue-rotate(263deg) brightness(113%) contrast(100%);
        transition: fill .7s ease
      }

    `	
  };
      
  static get properties() {
    return {
      hash: { type: Model },
      current: { type: String },
      tabs: { type: Array }
    };
  }

  constructor() {
    super();

    this.hash = HASH;
    this.current = '';
    this.tabs = [];
    Array.from(this.children).forEach(node => {
      if (node.localName == "dw-tab") {
        this.tabs.push(node);
      }
    });
  }

  renderLinks() {
    let links = [];
    let currentTab = this.current;
    if (currentTab == '' && this.tabs.length > 0) {
      currentTab = this.tabs[0].name;
    }

    for (let idx = 0; idx < this.tabs.length; ++idx) {
      let code = this.hash.get("code")
      if (this.tabs[idx].code && this.tabs[idx].code !== code) {
        continue
      }

      let classes = "";
      if (this.tabs[idx].name === currentTab) {
        this.hash.tab = this.tabs[idx].tabHashName();
        classes += " selected";
      }

      let addendum = code ? `?code=${code}` : ""

      // todo: this is literally the most deeply nesed usage of the turnary operator I have ever seen, it should probably die
      links.push(html`
          <a href="#${this.tabs[idx].tabHashName()}${addendum}" class="${classes}">
            <div class="selected_stripe"></div>
            <div class="label">
              <div class="icon">
                 ${(this.tabs[idx].name === "Dashboard") ? html`<img alt="dashboard logo" class="tabLogo" src="../images/svgs/dashboard.svg"><defs><style>.cls-1{fill:#fff;}</style></defs><title>hosts</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"></g></g></img>`
        : (this.tabs[idx].name === "Hosts") ? html`<img alt="hosts logo" class="tabLogo" src="../images/svgs/hosts.svg"><defs><style>.cls-1{fill:#fff;}</style></defs><title>hosts</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"></g></g></img>`
          : (this.tabs[idx].name === "Projects") ? html`<img alt="projects logo" class="tabLogo" src="../images/svgs/projects.svg"><defs><style>.cls-1{fill:#fff;}</style></defs><title>hosts</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"></g></g></img>`
            : (this.tabs[idx].name === "Mappings") ? html`<img alt="mappings logo" class="tabLogo" src="../images/svgs/mappings.svg"><defs><style>.cls-1{fill:#fff;}</style></defs><title>hosts</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"></g></g></img>`
              : (this.tabs[idx].name === "Rate Limits") ? html`<img alt="ratelimits logo" class="tabLogo" src="../images/svgs/ratelimits.svg"><defs><style>.cls-1{fill:#fff;}</style></defs><title>hosts</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"></g></g></img>`
                : (this.tabs[idx].name === "Plugins") ? html`<img alt="plugins logo" class="tabLogo" src="../images/svgs/plugins.svg"><defs><style>.cls-1{fill:#fff;}</style></defs><title>hosts</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"></g></g></img>`
                  : (this.tabs[idx].name === "Resolvers") ? html`<img alt="resolvers logo" class="tabLogo" src="../images/svgs/resolvers.svg"><defs><style>.cls-1{fill:#fff;}</style></defs><title>hosts</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"></g></g></img>`
                    : (this.tabs[idx].name === "Debugging") ? html`<img alt="debugging logo" class="tabLogo" src="../images/svgs/debugging.svg"><defs><style>.cls-1{fill:#fff;}</style></defs><title>hosts</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"></g></g></img>`
                      : (this.tabs[idx].name === "APIs") ? html`<img alt="api logo" class="tabLogo" src="../images/svgs/apis.svg"><defs><style>.cls-1{fill:#fff;}</style></defs><title>hosts</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"></g></g></img>`
                        : (this.tabs[idx].name === "Documentation") ? html`<img alt="docs logo" class="tabLogo" src="../images/svgs/docs.svg"><defs><style>.cls-1{fill:#fff;}</style></defs><title>hosts</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"></g></g></img>`
                          : (this.tabs[idx].name === "Support") ? html`<img alt="support logo" class="tabLogo" src="../images/svgs/support.svg"><defs><style>.cls-1{fill:#fff;}</style></defs><title>hosts</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"></g></g></img>`
                            : (this.tabs[idx].name === "Filters") ? html`<img alt="filters logo" class="tabLogo" src="../images/svgs/filters.svg"><defs><style>.cls-1{fill:#fff;}</style></defs><title>hosts</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"></g></g></img>`
                              : (this.tabs[idx].name === "YAML Download") ? html`<img alt="yaml logo" class="tabLogo" src="../images/svgs/yaml-downloads2.svg"><defs><style>.cls-1{fill:#fff;}</style></defs><title>hosts</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"></g></g></img>`
                              : html``}
              </div>
              <div class="name">${this.tabs[idx].tabName()}</div>
            </div>
          </a>
        `);
    }
    return links;
  }

  render() {
    this.tabs.forEach(tab => {
      if (this.hash.tab === tab.tabHashName()) {
        this.current = tab.name;
      }
    })

    let tabName = this.current;
    if (tabName === '') {
      tabName = this.tabs[0].name;
    }

    return html`
      <div class="col_wrapper">
        <div class="col_left">
          <div class="logo"><img src="../images/ambassador-logo-white.svg"/></div>
          <navigation>
            ${this.renderLinks()}
          </navigation>
        </div>
        <div class="col_right">
          <div class="content">
            <slot name="${tabName}"></slot>
          </div>
        </div>
      </div>
    `;
  }

}

customElements.define('dw-tabs', Tabs);

export class Tab extends View {
  static get properties() {
    return {
      _name: { type: String },
      icon: { type: String },
      hashname: { type: String },
      hash: { type: Model },
      code: { type: String }
    }
  }

  constructor() {
    super();

    this.icon = "";
    this._name = (this.getAttribute('slot') || '');
    this.updateHashname();
    this.hash = HASH;
  }

  updateHashname() {
    this.hashname = this.name.toLowerCase().replace(/[^\w]+/, '-');
  }

  set slot(val) {
    this._name = val;
    this.updateHashname();
  }

  set name(val) {
    this._name = val;
    this.updateHashname();
  }

  get name() {
    return this._name;
  }

  tabName() {
    return this._name;
  }

  tabIconSVG() {
    return this.icon;
  }

  tabHashName() {
    return this.hashname;
  }

  render() {
    let code = this.hash.get("code")
    if (!this.code || code === this.code) {
      return html`<slot></slot>`
    } else {
      return html``
    }
  }
}

customElements.define('dw-tab', Tab);
