/*
 * ResourceCollectionView
 * A LitElement subclass that implements a generic view on a sortable list of Views.
 * This class listens to a ResourceCollection, and adds/removes views on those Resources
 * as needed.
 */

/* Debug flag */
import { enableMVC } from "./utilities.js"

/* LitElement superclass. */
import { LitElement, html, css } from '../../vendor/lit-element.min.js'

export class ResourceCollectionView extends LitElement {

  /* properties()
   * These are the properties of the ResourceCollectionView. LitElement manages these declared properties and
   * provides various services depending on how they are used.  For further details on LitElement, see
   * https://lit-element.polymer-project.org/guide/properties
   */

  static get properties() {
    return {
      addState: {type: Boolean},
      sortFields: {type: Array},
      sortBy: {type: String}
    };
  }

  /* styles
   * These are the styles of the ResourceCollectionView. LitElement allows each Element to provide
   * additional css style specifications that are valid only for that LitElement.
   */

  static get styles() {
    return css`
      div.sortby {
          text-align: right;
      }
      div.sortby select {
        font-size: 0.85rem;
        border: 2px #c8c8c8 solid;
        text-transform: none; 
      }
      div.sortby select:hover {
        color: #5f3eff;
        transition: all .2s ease;
        border: 2px #5f3eff solid;
      }
    `
  }

  /* constructor()
   * model is the ResourceCollection that is being rendered by this ResourceCollectionView.
   */

  constructor() {
    super();

    /* sortFields is an array of {value: label} objects, where the value is the Resource property
     * on which to sort, and label is the display name for the HTML component.  For example, to allow
     * the option to sort by Resource name only, namespace, and hostname, set sortFields to:
     * [
     *  {label: "Resource Name", value: "name"},
     *  {label: "Namespace", value: "namespace"},
     *  {label: "Hostname", value: "hostname"},
     * ];
     */
    this.sortFields = [ {label: "Resource Name", value: "name"},
                        {label: "Namespace", value: "namespace"},
                        {label: "Hostname", value: "hostname"},];
    this.sortBy = "name";
    this.addState = "off";
  }

  onAdd() {
    throw Error("Not Yet Implemented");
  }

  onChangeSortByAttribute(e) {
    this.sortBy = e.target.options[e.target.selectedIndex].value;
  }

  /* onModelNotification.
  * This method is called for model-created notifications when a new Host has been created, and a
  * new view must be created to display that Host.
  */

  onModelNotification(model, message, parameter) {
    /* Create a new dw-host web component and add it as a child. Because this view is a web component, adding
     * that child component queues the appropriate re-render at the correct time,and are rendered in our <slot>.
    */
    if (message === 'created') {
      let viewClass = this.viewClass();
      let child_view = new viewClass(model);
      this.appendChild(child_view);
    }
  }

  /* readOnly()
   * If readOnly is set to false, then the collection can be added to by the user, e.g. a HostCollection, if
   * readOnly is false, will provide an Add button so that the end user can create a new Host and set its
   * attributes.
   *
   * if readOnly is set to true, then the collection's contents are only those Resources that are observed
   * in the snapshot, and there is no Add button or any mechanism for the user, from the Edge Admin UI, to
   * add new Resources.
   */
  readOnly() {
    return false;
  }

  /* render()
   * Render the list.  Key requirements:
   *   a) include the add-button
   *   b) include a single slot for where you want add to be
   *   c) include a slot for all the rest of the data
   */

  render() {
    if (enableMVC()) {
      return html`
        <div style="border:thick solid red">
            <link rel="stylesheet" href="../styles/resources.css">
            <div class="header_con">
                <div class="col">
                    <img alt="hosts logo" class="logo" src="../images/svgs/hosts.svg">
                        <defs><style>.cls-1{fill:#fff;}</style></defs>
                        <g id="Layer_2" data-name="Layer 2">
                            <g id="Layer_1-2" data-name="Layer 1">
                            </g>
                        </g>
                    </img>
                </div>
                
                <div class="col">
                    <h1>Hosts</h1>
                        <p>Hosts are domains that are managed by Ambassador Edge Stack, e.g., example.org</p>
                </div>
                
                 <div class="col2">
                    <a class="cta add ${this.readOnly() ? "off" : ""}" @click=${this.onAdd.bind(this)}>
                      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 30 30"><defs><style>.cls-a{fill:none;stroke:#000;stroke-linecap:square;stroke-miterlimit:10;stroke-width:2px;}</style></defs><title>add_1</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><line class="cls-a" x1="15" y1="9" x2="15" y2="21"/><line class="cls-a" x1="9" y1="15" x2="21" y2="15"/><circle class="cls-a" cx="15" cy="15" r="14"/></g></g></svg>
                      <div class="label">add</div>
                    </a>
 
                    <div class="sortby" >
                    <select id="sortByAttribute" @change=${this.onChangeSortByAttribute.bind(this)}>
                        ${this.sortFields.map(f => {return html`<option value="${f.value}">${f.label}</option>`})}
                    </select>
                </div>
 
            </div>
            </div>
            <slot name="add"></slot>
            <slot></slot>
        </div>`
    } else {
      return html``
    }
  }
}
