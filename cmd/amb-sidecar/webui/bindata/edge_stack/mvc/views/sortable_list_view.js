export class SortableModelListView extends ModelListView {

  // internal
  static get properties() {
    return {
      sortFields: { type: Array },
      sortBy: { type: String }
    };
  }

  /**
   * @param sortFields: A non-empty Array of {value, label} Objects by which it is possible to sort the ResourceSet. eg.:
   *   super([
   *     {
   *       value: "name",         // String. When selected, the value will be passed as argument in `sortFn`.
   *       label: "Mapping Name"  // String. Display label for the HTML component.
   *     },
   *     ...
   *   ]);
   */
  constructor(sortFields) {
    super();
    if (!sortFields || sortFields.length === 0) {
      throw new Error('please pass `sortFields` to constructor');
    }
    this.sortFields = sortFields;
    this.sortBy = this.sortFields[0].value;
  }

  onChangeSortByAttribute(e) {
    this.sortBy = e.target.options[e.target.selectedIndex].value;
  }

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

  render() {
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
    throw new Error("please implement renderSet()");
  }

  sortFn(sortByAttribute) {
    throw new Error("please implement sortFn(sortByAttribute)");
  }
}
