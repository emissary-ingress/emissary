/*
 * CollectionView
 * A LitElement subclass that implements a generic view on a sortable list of Views.
 * This class listens to a Collection of Resources, and adds/removes views on those Resources
 * as needed.
 */

/* Map merge operation */
import { mapMerge } from "./map.js"

/* LitElement superclass. */
import { LitElement, html } from '../../vendor/lit-element.min.js'

export class CollectionView extends LitElement {

  /* properties()
   * These are the properties of the CollectionView. LitElement manages these declared properties and
   * provides various services depending on how they are used.  For further details on LitElement, see
   * https://lit-element.polymer-project.org/guide/properties
   */

  static get properties() {
    return {
      sortFields: { type: Array },
      sortBy:     { type: String }
    };
  }

  /* styles
   * These are the styles of the CollectionView. LitElement allows each Element to provide
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
        text-transform: uppercase; 
      }
      div.sortby select:hover {
        color: #5f3eff;
        transition: all .2s ease;
        border: 2px #5f3eff solid;
      }
    `
  }
  /* constructor(model, sortFields)
   * model is the Collection that is being rendered by this CollectionView.
   * sortFields is an array of {value: label} objects, where the value is the Resource property
   * on which to sort, and label is the display name for the HTML component.
   */

  constructor(model, sortFields) {
    super();

    if (!sortFields || sortFields.length === 0) {
      throw new Error('please pass `sortFields` to new CollectionView()');
    }
    this.sortFields = sortFields;
    this.sortBy     = this.sortFields[0].value;
    this.addState   = "off";

    /* Add this as a listener to the model (a Collection of Resources) */
    model.addListener(this);
  }

  onChangeSortByAttribute(e) {
    this.sortBy = e.target.options[e.target.selectedIndex].value;
  }

  /* render()
   * Render the list.  Key requirements:
   *   a) include the add-button
   *   b) include a single slot for where you want add to be
   *   c) include a slot for all the rest of the data
   */

  render() {
    return html`
        <link rel="stylesheet" href="../styles/resources.css">
        <add-button @click=${this.onAdd.bind(this)}></add-button>
                 <slot name="add"></slot>
                 <slot></slot>`
  }

  renderSorted() {
    return html`
      <div class="sortby">Sort by
        <select id="sortByAttribute" @change=${this.onChangeSortByAttribute}>
          ${this.sortFields.map(f => {
      return html`<option value="${f.value}">${f.label}</option>`
    })}
        </select>
      </div>
      ${this.resources.sort(this.sortFn(this.sortBy)) && this.renderSet()}`
  }

  renderSet() {
    throw new Error("please implement ${this.constructor.name}.renderSet()");
  }

  sortFn(sortByAttribute) {
    throw new Error("please implement ${this.constructor.name}.sortFn(sortByAttribute)");
  }
}

