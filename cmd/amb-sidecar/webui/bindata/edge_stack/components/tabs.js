import { LitElement, html, svg, css } from '../vendor/lit-element.min.js'
import {useContext} from './context.js';

/**
 * Provides a small wrapper around named slots, to properly
 * render one of a series of "tabs".
 */
export class Tabs extends LitElement {
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
        height: 70px;
        transition: all .9s ease
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
      
      navigation a .label .icon svg, navigation a.selected .label .icon svg {
        width: 100%;
        height: auto;
        max-height: 35px
      }
      
      navigation a .label .icon svg path, navigation a .label .icon svg polygon, navigation a .label .icon svg rect {
        fill: #9a9a9a;
        transition: fill .7s ease
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
    `	
  };
      
  static get properties() {
    return {
      current: { type: String },
      tabs: { type: Array }
    };
  }

  constructor() {
    super();

    this.current = '';
    this.tabs = [];
    Array.from(this.children).forEach(node => {
      if (node.localName == "dw-tab") {
        this.tabs.push(node);
      }
    });
  }

  handleHashChange() {
    for (let idx = 0; idx < this.tabs.length; idx++) {
      if(window.location.hash === ('#' + this.tabs[idx].tabHashName())) {
        this.current = this.tabs[idx].name;
        break;
      }
    }
  }

  connectedCallback() {
    super.connectedCallback();
    window.addEventListener("hashchange", this.handleHashChange.bind(this), false);
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    window.remmoveEventListener("hashchange", this.handleHashChange.bind(this), false);
  }

  renderLinks() {
    let links = [];
    let currentTab = this.current;
    if (currentTab == '' && this.tabs.length > 0) {
      currentTab = this.tabs[0].name;
    }

    for (let idx = 0; idx < this.tabs.length; ++idx) {
      let classes = "";
      if (this.tabs[idx].name === currentTab) {
        if (window.location.hash.length > 300) {
          /* if a long hash, then it might be a login cookie */
          if (useContext('auth-state', null)[0]) {
            /* logged in, so don't need to preserve the hash tag */
            window.location.hash = "#" + this.tabs[idx].tabHashName();
          } else {
            /* not logged in, so don't change the hash cookie */
          }
        } else {
          /* shorter hash, so it can't be a login cookie, so go ahead and change it */
          window.location.hash = "#" + this.tabs[idx].tabHashName();
        }

        classes += " selected";
      }

      //TODO need to fix the duplicate embedding of the svg icon here and then again in each tab detail page.
      //  We really just want the svg once, perhaps in the index.html creation of the dw-tab and then have
      //  that used by both the tab system here and the detail page.
      links.push(html`
          <a href="#${this.tabs[idx].tabHashName()}" class="${classes}">
            <div class="selected_stripe"></div>
            <div class="label">
              <div class="icon">
                 ${(this.tabs[idx].name === "Dashboard") ? svg`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24.44 18.8"><defs><style>.cls-1{fill:#fff;}</style></defs><title>dashboard</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M12.22,0A12.23,12.23,0,0,0,1.78,18.58a.47.47,0,0,0,.65.15.46.46,0,0,0,.15-.64,11.28,11.28,0,1,1,19.28,0,.46.46,0,0,0,.15.64.45.45,0,0,0,.25.07.47.47,0,0,0,.4-.22A12.23,12.23,0,0,0,12.22,0Zm7.66,6.25-6.37,6.36a2.28,2.28,0,0,0-1.29-.39,2.35,2.35,0,0,0-2.35,2.35,2.28,2.28,0,0,0,.39,1.29L8.13,18a.45.45,0,0,0,0,.66.46.46,0,0,0,.66,0l2.14-2.13a2.35,2.35,0,0,0,3.64-2,2.28,2.28,0,0,0-.39-1.29l6.36-6.37a.47.47,0,1,0-.66-.66ZM12.22,16a1.41,1.41,0,1,1,1.41-1.41A1.41,1.41,0,0,1,12.22,16Zm0-12.22A8.38,8.38,0,0,1,17.9,6a.47.47,0,1,0,.64-.7h0A9.4,9.4,0,0,0,4.19,17.11a.48.48,0,0,0,.4.23.55.55,0,0,0,.25-.07A.47.47,0,0,0,5,16.62,8.47,8.47,0,0,1,12.22,3.76Z"/></g></g></svg>`
        : (this.tabs[idx].name === "Hosts") ? svg`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 21.1 22.5"><defs><style>.cls-1{fill:#fff;}</style></defs><title>hosts</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M20.36,2,19.23.78A2.87,2.87,0,0,0,17.39,0H3.72A2.89,2.89,0,0,0,1.87.78L.75,2A2.94,2.94,0,0,0,0,3.8V7.57A1.43,1.43,0,0,0,1.43,9H19.68A1.43,1.43,0,0,0,21.1,7.57V3.8A3,3,0,0,0,20.36,2Zm-.44,5.62a.25.25,0,0,1-.24.25H1.43a.25.25,0,0,1-.25-.25V3.8a1.78,1.78,0,0,1,.42-1L2.72,1.6a1.67,1.67,0,0,1,1-.42H17.39a1.67,1.67,0,0,1,1,.42l1.13,1.17a1.84,1.84,0,0,1,.41,1Zm-.24,2.9H1.43A1.43,1.43,0,0,0,0,11.9v3.78a1.43,1.43,0,0,0,1.43,1.43H19.68a1.43,1.43,0,0,0,1.42-1.43V11.9A1.43,1.43,0,0,0,19.68,10.47Zm.24,5.21a.25.25,0,0,1-.24.25H1.43a.25.25,0,0,1-.25-.25V11.9a.25.25,0,0,1,.25-.25H19.68a.25.25,0,0,1,.24.25ZM17.21,4.89a.79.79,0,1,0,0,1.58.79.79,0,0,0,0-1.58ZM14.39,13a.79.79,0,0,0-.79.8.79.79,0,0,0,.79.79.79.79,0,0,0,.8-.79A.8.8,0,0,0,14.39,13ZM11.7,13a.8.8,0,0,0-.8.8.8.8,0,1,0,.8-.8Zm5.38,0a.8.8,0,0,0-.79.8.8.8,0,0,0,1.59,0A.8.8,0,0,0,17.08,13Zm-2.72-8.1a.79.79,0,1,0,0,1.58.79.79,0,1,0,0-1.58ZM19,19.75H12.64a2.17,2.17,0,0,0-4.17,0H2.12a.58.58,0,0,0-.59.59.59.59,0,0,0,.59.59H8.47a2.17,2.17,0,0,0,4.17,0H19a.59.59,0,0,0,.59-.59A.58.58,0,0,0,19,19.75Zm-7.44.59a1,1,0,1,1-1-1,1,1,0,0,1,1,1Z"/></g></g></svg>`
          : (this.tabs[idx].name === "Mappings") ? svg`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 25.06 23.27"><defs><style>.cls-m{fill:#fff;}</style></defs><title>mappings</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-m" d="M25,19.86l-.81-1.62V14.32a.45.45,0,0,0-.45-.45H21.48V12.08a.45.45,0,0,0-.45-.45H14.25A1.8,1.8,0,0,0,13,10.35V8c1.84-.08,3.57-.69,3.57-1.78V4.47a.39.39,0,0,0,0-.18.75.75,0,0,0,0-.26V2.24a.41.41,0,0,0,0-.19.75.75,0,0,0,0-.26c0-1.18-2-1.79-4-1.79s-4,.61-4,1.79a1.09,1.09,0,0,0,0,.26.58.58,0,0,0,0,.19V4a1.09,1.09,0,0,0,0,.26.58.58,0,0,0,0,.18V6.26C8.5,7.35,10.24,8,12.08,8v2.31a1.8,1.8,0,0,0-1.28,1.28H4a.45.45,0,0,0-.45.45v1.79H1.34a.45.45,0,0,0-.45.45v3.92L.05,19.93a.46.46,0,0,0,0,.44.44.44,0,0,0,.38.21H7.61a.45.45,0,0,0,.45-.45A.44.44,0,0,0,8,19.86l-.81-1.62V14.32a.45.45,0,0,0-.45-.45H4.47V12.53H10.8a1.8,1.8,0,0,0,1.28,1.28v2.3H9.4a.45.45,0,0,0-.45.44v4.37l-.85,1.7a.45.45,0,0,0,.4.65h8.06a.45.45,0,0,0,.45-.45.42.42,0,0,0-.09-.27l-.81-1.63V16.55a.45.45,0,0,0-.45-.44H13v-2.3a1.8,1.8,0,0,0,1.27-1.28h6.33v1.34H18.34a.45.45,0,0,0-.44.45v3.92l-.85,1.69a.46.46,0,0,0,0,.44.44.44,0,0,0,.38.21h7.16a.45.45,0,0,0,.45-.45A.44.44,0,0,0,25,19.86Zm-23.8-.17.45-.9H6.43l.45.9ZM6.26,17.9H1.79V14.76H6.26Zm3,4.47.45-.89h5.71l.45.89Zm6-1.79H9.84V17h5.37ZM9.4,3a7,7,0,0,0,3.13.63A7,7,0,0,0,15.66,3V4c0,.26-1.1.89-3.13.89S9.4,4.29,9.4,4ZM12.53.89c2,0,3.13.64,3.13.9s-1.1.89-3.13.89S9.4,2.05,9.4,1.79,10.49.89,12.53.89ZM9.4,6.26V5.19a7.09,7.09,0,0,0,3.13.63,7.15,7.15,0,0,0,3.13-.63V6.26c0,.26-1.1.9-3.13.9S9.4,6.52,9.4,6.26ZM12.53,13a.9.9,0,1,1,0-1.79.9.9,0,0,1,0,1.79Zm6.26,1.78h4.48V17.9H18.79Zm-.62,4.93.45-.9h4.82l.44.9Z"/></g></g></svg>`
            : (this.tabs[idx].name === "Rate Limits") ? svg`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 23.74 21.9"><defs><style>.cls-1{fill:#fff;}</style></defs><title>Rate</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M23.09,20.54H.66a.69.69,0,0,0-.66.7.66.66,0,0,0,.66.66H23.09a.7.7,0,0,0,.65-.71A.65.65,0,0,0,23.09,20.54Z"/><path class="cls-1" d="M23.09,5.14H18.6a.66.66,0,0,0-.66.66V9.34H14.77V.66A.65.65,0,0,0,14.11,0H9.63A.66.66,0,0,0,9,.66V11.21H5.8V9.07a.66.66,0,0,0-.66-.66H.66A.66.66,0,0,0,0,9.07v8.41a.66.66,0,0,0,.66.66H23.09a.66.66,0,0,0,.65-.66V5.8A.65.65,0,0,0,23.09,5.14ZM1.32,9.73H4.49v7.09H1.32Zm4.48,2.8H9v4.29H5.8Zm4.49-.66V1.32h3.17v15.5H10.29Zm4.48-1.21h3.17v6.16H14.77Zm7.66,6.16H19.26V6.46h3.17Z"/></g></g></svg>`
              : (this.tabs[idx].name === "Plugins") ? svg`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24.99 21.04"><defs><style>.cls-1{fill:#fff;}</style></defs><title>services</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M0,0V21H25V0ZM5.94,14.4l.94-.94,1.85,1.32L9,14.63a6.39,6.39,0,0,1,2.23-.92l.27-.06.37-2.24h1.34l.36,2.24.27.06a6.39,6.39,0,0,1,2.23.92l.23.15,1.85-1.33.94,1-1.32,1.85.15.23a6.39,6.39,0,0,1,.92,2.23l.06.27,2.24.36v.89H3.89v-.89L6.13,19l.06-.27a6.39,6.39,0,0,1,.92-2.23l.15-.23Zm18.24,5.83H21.91v-1a.7.7,0,0,0-.58-.69l-1.8-.29a7.12,7.12,0,0,0-.83-2l1.06-1.47a.7.7,0,0,0-.07-.9l-1.08-1.08a.7.7,0,0,0-.9-.07l-1.47,1.06a7.12,7.12,0,0,0-2-.83L14,11.18a.7.7,0,0,0-.68-.58H11.73a.69.69,0,0,0-.68.58l-.3,1.8a7.12,7.12,0,0,0-2,.83L7.27,12.74a.69.69,0,0,0-.89.07L5.29,13.9a.71.71,0,0,0-.07.89l1.07,1.48a7.12,7.12,0,0,0-.83,2l-1.79.29a.71.71,0,0,0-.59.69v1H.81V5.2H24.18ZM.81,4.4V.81H24.18V4.4Z"/><rect class="cls-1" x="2.15" y="2.2" width="1.33" height="0.81"/><rect class="cls-1" x="4.01" y="2.2" width="1.33" height="0.81"/><rect class="cls-1" x="5.88" y="2.2" width="1.33" height="0.81"/><path class="cls-1" d="M12.5,17a3.23,3.23,0,0,1,3.22,3.23h.81a4,4,0,0,0-8.07,0h.81A3.23,3.23,0,0,1,12.5,17Z"/></g></g></svg>`
                : (this.tabs[idx].name === "Resolvers") ? svg`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 25.82 25.82"><defs><style>.cls-1{fill:#fff;}</style></defs><title>resolve</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M12.91,4.19a8.72,8.72,0,1,0,8.72,8.72A8.73,8.73,0,0,0,12.91,4.19ZM20,9.53H16.58a9.53,9.53,0,0,0-1.72-4.25A7.9,7.9,0,0,1,20,9.53Zm-4,3.38a19.91,19.91,0,0,1-.17,2.54H9.94a19.06,19.06,0,0,1,0-5.08h5.94A19.91,19.91,0,0,1,16.05,12.91ZM12.91,5c1.11,0,2.28,1.72,2.82,4.5H10.09C10.63,6.75,11.8,5,12.91,5ZM11,5.28A9.53,9.53,0,0,0,9.24,9.53H5.8A7.9,7.9,0,0,1,11,5.28ZM5,12.91a7.65,7.65,0,0,1,.43-2.54H9.1a19.06,19.06,0,0,0,0,5.08H5.46A7.65,7.65,0,0,1,5,12.91Zm.77,3.38H9.24A9.53,9.53,0,0,0,11,20.54,7.9,7.9,0,0,1,5.8,16.29Zm7.11,4.5c-1.11,0-2.28-1.72-2.82-4.5h5.64C15.19,19.07,14,20.79,12.91,20.79Zm1.95-.25a9.53,9.53,0,0,0,1.72-4.25H20A7.9,7.9,0,0,1,14.86,20.54Zm1.86-5.09a18.67,18.67,0,0,0,.17-2.54,18.67,18.67,0,0,0-.17-2.54h3.64a7.72,7.72,0,0,1,0,5.08Z"/><path class="cls-1" d="M16,24.47a12,12,0,0,1-11.57-20l.67-.66v.33a.47.47,0,1,0,.94,0V2.64a.44.44,0,0,0,0-.15l0-.06a.5.5,0,0,0-.08-.12h0a.38.38,0,0,0-.13-.09.23.23,0,0,0-.11,0l-.07,0H4.11a.47.47,0,0,0-.47.47.48.48,0,0,0,.47.48h.33l-.66.66A12.92,12.92,0,0,0,3.78,22a13,13,0,0,0,9.15,3.78,12.83,12.83,0,0,0,3.32-.44.46.46,0,0,0,.35-.45.45.45,0,0,0,0-.12A.49.49,0,0,0,16,24.47Z"/><path class="cls-1" d="M21.71,22.7h-.33L22,22A12.92,12.92,0,0,0,22,3.78,12.93,12.93,0,0,0,9.57.44a.46.46,0,0,0-.35.45.45.45,0,0,0,0,.12.48.48,0,0,0,.58.34,12,12,0,0,1,11.57,20l-.67.66v-.33a.47.47,0,0,0-.94,0v1.47a.49.49,0,0,0,0,.15l0,.06a.5.5,0,0,0,.08.12.36.36,0,0,0,.12.08l.06,0,.15,0h1.47a.47.47,0,0,0,.47-.47A.48.48,0,0,0,21.71,22.7Z"/></g></g></svg>`
                  : (this.tabs[idx].name === "Debugging") ? svg`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 23.78 23.37"><defs><style>.cls-1{fill:#fff;}</style></defs><title>debug</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M23.16,13.94H20a9.24,9.24,0,0,0-1.73-4.85l2.29-2.28a.62.62,0,0,0-.88-.88L17.43,8.14a8.86,8.86,0,0,0-1.09-.94,3.35,3.35,0,0,0,.1-.87,4.57,4.57,0,0,0-2.37-4,4,4,0,0,1,1.68-1.12.62.62,0,0,0,.4-.78.61.61,0,0,0-.78-.4,5.27,5.27,0,0,0-2.53,1.85,4.56,4.56,0,0,0-1.9,0A5.31,5.31,0,0,0,8.41,0a.62.62,0,0,0-.78.4A.62.62,0,0,0,8,1.21,3.92,3.92,0,0,1,9.71,2.33a4.56,4.56,0,0,0-2.38,4,3.34,3.34,0,0,0,.11.87,8.18,8.18,0,0,0-1.09.94L4.14,5.93a.62.62,0,1,0-.88.88L5.55,9.09a9.33,9.33,0,0,0-1.74,4.85H.62a.62.62,0,0,0,0,1.24h3.2A9.1,9.1,0,0,0,5.55,20L3.26,22.31a.62.62,0,0,0,0,.87.6.6,0,0,0,.88,0L6.35,21a7.75,7.75,0,0,0,5.54,2.4A7.75,7.75,0,0,0,17.43,21l2.21,2.21a.63.63,0,0,0,.44.19.65.65,0,0,0,.44-.19.62.62,0,0,0,0-.87L18.23,20A9.1,9.1,0,0,0,20,15.18h3.2a.62.62,0,0,0,0-1.24ZM11.89,3a3.32,3.32,0,0,1,3.32,3.31,1.37,1.37,0,0,1-.3,1c-.42.42-1.5.41-2.63.41H11.5c-1.14,0-2.21,0-2.63-.41a1.37,1.37,0,0,1-.3-1A3.32,3.32,0,0,1,11.89,3ZM5,14.56a7.83,7.83,0,0,1,3-6.29C8.82,9,10,9,11.27,9V22.09C7.78,21.75,5,18.5,5,14.56Zm7.48,7.53V9h.13a4.51,4.51,0,0,0,3.07-.71,7.85,7.85,0,0,1,3,6.29C18.75,18.5,16,21.75,12.51,22.09Z"/></g></g></svg>`
                    : (this.tabs[idx].name === "APIs") ? svg`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24.02 23.53"><title>apis</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M12,5.19a6.58,6.58,0,1,0,6.58,6.58A6.58,6.58,0,0,0,12,5.19Zm0,12.37a5.8,5.8,0,1,1,5.8-5.79A5.8,5.8,0,0,1,12,17.56Z"/><polygon class="cls-1" points="9.68 9.47 7.28 11.88 9.68 14.28 10.23 13.73 8.38 11.88 10.23 10.02 9.68 9.47"/><polygon class="cls-1" points="13.79 10.02 15.65 11.88 13.79 13.73 14.35 14.28 16.75 11.88 14.35 9.47 13.79 10.02"/><rect class="cls-1" x="9.09" y="11.38" width="5.84" height="0.78" transform="translate(-2.74 19.84) rotate(-73.09)"/><path class="cls-1" d="M23.39,10.05l-2.34-.38A8.8,8.8,0,0,0,20,7l1.4-1.94a.76.76,0,0,0-.08-1L19.89,2.76a.75.75,0,0,0-1-.08L17,4.07A9.1,9.1,0,0,0,14.35,3L14,.64A.75.75,0,0,0,13.22,0H10.8a.75.75,0,0,0-.75.63L9.67,3A9,9,0,0,0,7,4.07L5.11,2.68a.76.76,0,0,0-1,.08L2.76,4.13a.77.77,0,0,0-.08,1L4.07,7A9,9,0,0,0,3,9.67l-2.35.38a.75.75,0,0,0-.63.74v2a.75.75,0,0,0,.63.74L3,13.86A9.21,9.21,0,0,0,4.07,16.5L2.68,18.43a.75.75,0,0,0,.08,1l1.38,1.38a.74.74,0,0,0,1,.07L7,19.46a9.06,9.06,0,0,0,2.63,1.1l.38,2.34a.76.76,0,0,0,.75.63h2.43A.74.74,0,0,0,14,22.9l.38-2.34A9.12,9.12,0,0,0,17,19.46l1.93,1.39a.76.76,0,0,0,1-.08l1.36-1.37a.75.75,0,0,0,.08-1L20,16.5a9,9,0,0,0,1.1-2.64l2.34-.38a.74.74,0,0,0,.63-.74v-2A.74.74,0,0,0,23.39,10.05Zm-3,3.13-.06.26a8.12,8.12,0,0,1-1.19,2.85l-.14.23,1.7,2.33-1.34,1.36L17,18.52l-.22.14a8.16,8.16,0,0,1-2.86,1.19l-.26,0-.44,2.85-2.41,0-.46-2.87-.27,0a8.19,8.19,0,0,1-2.85-1.19L7,18.52l-2.32,1.7L3.31,18.89,5,16.52l-.15-.23a8.12,8.12,0,0,1-1.19-2.85l-.05-.26L.78,12.74l0-1.92,2.87-.46.05-.27A8.19,8.19,0,0,1,4.87,7.24L5,7,3.32,4.68,4.65,3.32,7,5l.23-.15a8.19,8.19,0,0,1,2.85-1.19l.27-.05L10.8.78l2.4,0,.47,2.87.26.05a8.16,8.16,0,0,1,2.86,1.19L17,5l2.33-1.71,1.37,1.34L19,7l.14.23a8.19,8.19,0,0,1,1.19,2.85l.06.27,2.84.43,0,1.92Z"/></g></g></svg>`
                      : (this.tabs[idx].name === "Documentation") ? svg`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16.84 24.48"><defs><style>.cls-1{fill:#fff;}</style></defs><title>documents</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M16.52,3,5.24,0a.45.45,0,0,0-.37.08.42.42,0,0,0-.16.33v1.8L2.84,1.74a.41.41,0,0,0-.37.07.42.42,0,0,0-.16.33V4.4L.53,3.94A.41.41,0,0,0,.16,4,.42.42,0,0,0,0,4.34V21.12a.42.42,0,0,0,.31.4L11.6,24.46l.11,0a.38.38,0,0,0,.25-.09.43.43,0,0,0,.17-.33V21.8l1.78.46.11,0a.38.38,0,0,0,.25-.09.4.4,0,0,0,.17-.33V20.05l1.87.49.1,0a.39.39,0,0,0,.26-.09.43.43,0,0,0,.17-.33V3.36A.43.43,0,0,0,16.52,3ZM11.28,23.51.84,20.79V4.89L11.28,7.61Zm2.32-2.2-1.47-.38V7.28a.42.42,0,0,0-.32-.4L3.15,4.62V2.69L13.6,5.41ZM16,19.59l-1.55-.4V5.08a.42.42,0,0,0-.32-.4L5.55,2.44V1L16,3.69Z"/><path class="cls-1" d="M2.39,8.94,9.51,11a.2.2,0,0,0,.11,0,.44.44,0,0,0,.41-.3.43.43,0,0,0-.29-.52L2.62,8.13a.42.42,0,1,0-.23.81Z"/><path class="cls-1" d="M2.39,12.16l7.12,2.07h.11a.44.44,0,0,0,.41-.3.43.43,0,0,0-.29-.52L2.62,11.35a.43.43,0,0,0-.52.29A.42.42,0,0,0,2.39,12.16Z"/><path class="cls-1" d="M2.39,15.38l7.12,2.07.11,0a.42.42,0,0,0,.12-.83L2.62,14.58a.41.41,0,0,0-.52.28A.42.42,0,0,0,2.39,15.38Z"/><path class="cls-1" d="M2.39,18.6l7.12,2.07.11,0a.42.42,0,0,0,.12-.83L2.62,17.8a.42.42,0,1,0-.23.8Z"/></g></g></svg>`
                        : (this.tabs[idx].name === "Support") ? svg`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 25.13 20.81"><title>support</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><rect class="cls-1" x="14.3" y="6.09" width="1.18" height="0.89"/><rect class="cls-1" x="16.65" y="6.09" width="1.18" height="0.89"/><rect class="cls-1" x="19.01" y="6.09" width="1.18" height="0.89"/><path class="cls-2" d="M23.9.25H10.52a1,1,0,0,0-1,1V11.87a1,1,0,0,0,1,1h5.93l4.43,3.44a.55.55,0,0,0,.35.12.58.58,0,0,0,.57-.57v-3h2.1a1,1,0,0,0,1-1V1.23A1,1,0,0,0,23.9.25Zm.41,11.62a.41.41,0,0,1-.41.41H21.22v3.57l-4.58-3.57H10.52a.4.4,0,0,1-.4-.41V1.23a.4.4,0,0,1,.4-.4H23.9a.4.4,0,0,1,.41.4Z"/><path class="cls-2" d="M15,16a.41.41,0,0,1-.41.4H8.48L3.91,20V16.41H1.23a.4.4,0,0,1-.4-.4V5.37A.4.4,0,0,1,1.23,5H7.79V4.38H1.23a1,1,0,0,0-1,1V16a1,1,0,0,0,1,1H3.34v3a.56.56,0,0,0,.32.51.54.54,0,0,0,.25.06.55.55,0,0,0,.34-.13L8.68,17h5.93a1,1,0,0,0,1-1V14.36H15Z"/></g></g></svg>`
                          : (this.tabs[idx].name === "Filters") ? svg`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" width="32" height="32"><title>filter</title><g class="nc-icon-wrapper" stroke-linecap="square" stroke-linejoin="miter" stroke-width="2" fill="#608cee" stroke="#608cee"><polygon points="30 5 19 16 19 26 13 30 13 16 2 5 2 1 30 1 30 5" fill="none" stroke="#111111" stroke-miterlimit="10"/></g></svg>`
                            : (this.tabs[idx].name === "YAML Download") ? svg`<?xml version="1.0" encoding="utf-8"?>
<!-- Generator: Adobe Illustrator 24.0.0, SVG Export Plug-In . SVG Version: 6.00 Build 0)  -->
<svg version="1.1" id="Layer_1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" x="0px" y="0px"
         viewBox="0 0 24 24" style="enable-background:new 0 0 24 24;" xml:space="preserve">
<style type="text/css">
        .st0{fill:none;stroke:#000000;stroke-width:2;stroke-miterlimit:10;}
        .st1{fill:none;stroke:#000000;stroke-width:2;stroke-linecap:square;stroke-miterlimit:10;}
</style>
<title>move layer down</title>
<g>
        <line class="st0" x1="12" y1="1" x2="12" y2="17"/>
        <polyline class="st1" points="18,11 12,17 6,11  "/>
        <line class="st1" x1="22" y1="22" x2="2" y2="22"/>
</g>
</svg>`
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
    if (this.current === '') {
      let hash = window.location.hash.slice(1);
      this.tabs.forEach(tab => {
        if (hash === tab.tabHashName()) {
          this.current = tab.name;
        }
      })
    }

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

export class Tab extends LitElement {
  static get properties() {
    return {
      _name: { type: String },
      icon: { type: String },
      hashname: { type: String }
    }
  }

  constructor() {
    super();

    this.icon = "";
    this._name = (this.getAttribute('slot') || '');
    this.updateHashname();
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
    return html`<slot></slot>`;
  }
}

customElements.define('dw-tab', Tab);
