/**
 * ResourceCollectionView
 *
 * A LitElement subclass that implements a generic view on a sortable list of Views.  This class listens to a
 * ResourceCollection, and adds/removes views on those Resources as needed.
 */

/* LitElement superclass. */
import { html, css, repeat } from '../../vendor/lit-element.min.js'
import { View } from './view.js'

export class ResourceCollectionView extends View {

  /**
   * properties
   *
   * These are the properties of the ResourceCollectionView. LitElement manages these declared properties and
   * provides various services depending on how they are used.  For further details on LitElement, see
   * https://lit-element.polymer-project.org/guide/properties
   */

  static get properties() {
    return {
      model: {type: Object},
      sortFields: {type: Array},
      sortBy: {type: String}
    };
  }

  /**
   * styles
   *
   * These are the styles of the ResourceCollectionView. LitElement allows each Element to provide additional css style
   * specifications that are valid only for that LitElement.
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
      
      
      .header_con, .header_con .col {
        display: -webkit-flex;
        display: -ms-flexbox;
        display: flex;
        -webkit-justify-content: center;
        -ms-flex-pack: center;
        justify-content: center
      }
      
      .header_con {
        margin: 30px 0 0;
        -webkit-flex-direction: row;
        -ms-flex-direction: row;
        flex-direction: row
      }
      
      .header_con .col {
        -webkit-flex: 0 0 80px;
        -ms-flex: 0 0 80px;
        flex: 0 0 80px;
        -webkit-align-content: center;
        -ms-flex-line-pack: center;
        align-content: center;
        -webkit-align-self: center;
        -ms-flex-item-align: center;
        align-self: center;
        -webkit-flex-direction: column;
        -ms-flex-direction: column;
        flex-direction: column
      }
      
      .header_con .col svg {
        width: 100%;
        height: 60px
      }
      
      .header_con .col img {
        width: 100%;
        height: 60px;
      }
      
      .header_con .col img path {
        fill: #5f3eff
      }
      
      .header_con .col svg path {
        fill: #5f3eff
      }
      
      .header_con .col:nth-child(2) {
        -webkit-flex: 2 0 auto;
        -ms-flex: 2 0 auto;
        flex: 2 0 auto;
        padding-left: 20px
      }
      
      .header_con .col h1 {
        padding: 0;
        margin: 0;
        font-weight: 400
      }
      
      .header_con .col p {
        margin: 0;
        padding: 0
      }
      
      .header_con .col2, .col2 a.cta .label {
        -webkit-align-self: center;
        -ms-flex-item-align: center;
        -ms-grid-row-align: center;
        align-self: center
      }
      
      .logo {
        filter: invert(19%) sepia(64%) saturate(4904%) hue-rotate(248deg) brightness(107%) contrast(101%);
      }
      
      .col2 a.cta  {
        text-decoration: none;
        border: 2px #efefef solid;
        border-radius: 10px;
        width: 90px;
        padding: 6px 8px;
        max-height: 35px;
        -webkit-flex: auto;
        -ms-flex: auto;
        flex: auto;
        margin: 10px auto;
        color: #000;
        transition: all .2s ease;
        cursor: pointer;
      }
      
      .header_con .col2 a.cta  {
        border-color: #c8c8c8;
      }
      
      .col2 a.cta .label {
        text-transform: uppercase;
        font-size: .8rem;
        font-weight: 600;
        line-height: 1rem;
        padding: 0 0 0 10px;
        -webkit-flex: 1 0 auto;
        -ms-flex: 1 0 auto;
        flex: 1 0 auto
      }
      
      .col2 a.cta svg {
        width: 15px;
        height: auto
      }
      
      .col2 a.cta svg path, .col2 a.cta svg polygon {
        transition: fill .7s ease;
        fill: #000
      }
      
      .col2 a.cta:hover {
        color: #5f3eff;
        transition: all .2s ease;
        border: 2px #5f3eff solid
      }
      
      .col2 a.cta:hover svg path, .col2 a.cta:hover svg polygon {
        transition: fill .2s ease;
        fill: #5f3eff
      }
      
      .col2 a.cta {
        display: -webkit-flex;
        display: -ms-flexbox;
        display: flex;
        -webkit-flex-direction: row;
        -ms-flex-direction: row;
        flex-direction: row
      }
      
      .col2 a.off {
        display: none;
      }      
    `
  }

  /**
   * constructor()
   *
   * model is the ResourceCollection that is being rendered by this ResourceCollectionView.
   */
  constructor(model) {
    super();

    /* model is a ResourceCollection (e.g. HostCollection) */
    this.model = model;

    /* sortFields is an array of {value: label} objects, where the value is the Resource property
     * on which to sort, and label is the display name for the HTML component.  For example, to allow
     * the option to sort by Resource name only, namespace, and hostname, set sortFields to:
     * [
     *  {label: "Resource Name", value: "name"},
     *  {label: "Namespace", value: "namespace"},
     *  {label: "Hostname", value: "hostname"},
     * ];
     */
    this.sortFields = [ {label: "Name",      value: "name"},
                        {label: "Namespace", value: "namespace"},
                        {label: "Hostname",  value: "hostname"},];
    this.sortBy = "name";
  }

  /**
   * onAddButton()
   *
   * This method is called when the user has clicked on the Add button, to create a new Resource in the collection.
   */

  onAddButton() {
    this.model.new()
  }

  /**
   * onChangeSortByAttribute(event)
   *
   * This method is called when the user has selected an attribute for sorting the ResourceCollectionView.
   */
  onChangeSortByAttribute(event) {
    this.sortBy = event.target.options[event.target.selectedIndex].value;
  }

  /**
   * sortedResources()
   *
   * Returns the resources sorted in the order the user has selected.
   */
  sortedResources() {
    let result = Array.from(this.model);
    result.sort((r1, r2) => {
      /* any pending adds sort to the top of the list. */
      if (r1.isNew() && r1.isPending())
        return -1;

      if (r2.isNew() && r2.isPending())
        return 1;

      return r1[this.sortBy].localeCompare(r2[this.sortBy])
    })
    return result;
  }

  /**
   * readOnly()
   *
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

  /**
   * render()
   *
   * Render the list.  Key requirements:
   *   a) include the add-button
   *   b) include a single slot for where you want add to be
   *   c) include a slot for all the rest of the data
   */
  render() {
      let logoPair  = this.pageLogo();
      let logoText  = logoPair[0];
      let logoPath  = "../images/svgs/" + logoPair[1];
      let pageTitle = this.pageTitle();
      let description  = this.pageDescription();

      return html`
        <div>
            <div class="header_con">
                <div class="col">
                    <img alt=${logoText} class="logo" src=${logoPath}>
                        <defs><style>.cls-1{fill:#fff;}</style></defs>
                        <g id="Layer_2" data-name="Layer 2">
                            <g id="Layer_1-2" data-name="Layer 1">
                            </g>
                        </g>
                    </img>
                </div>
                
                <div class="col">
                    <h1>${pageTitle}</h1>
                        <p>${description}</p>
                </div>
                
                 <div class="col2">
                    <a class="cta add ${this.readOnly() ? "off" : ""}" @click=${this.onAddButton.bind(this)}>
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
            ${repeat(this.sortedResources(), (r)=>r.key(), this.renderResource.bind(this))}
        </div>`
  }

  /**
   * Override to customize how an individual resource is rendered.
   */
  renderResource(resource) {
    throw new Error("must implement renderResource()")
  }

  /**
   * sortMenu()
   *
   * Returns the element for the sort popup menu.  This is used for dynamically hiding and showing the element
   * depending on whether there are any views being displayed; if none, no need for sorting.
   */

  sortMenu() {
    return this.shadowRoot.getElementById("sortByAttribute");
  }

}
