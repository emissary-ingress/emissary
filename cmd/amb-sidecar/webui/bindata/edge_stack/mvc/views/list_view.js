/**
 * The Model class is an abstract base class that is extended in
 * order to create a container widget for listing any sort of Model.
 * Subclasses will extend this.
 *
 *   render() --> tell us how to display the collection
 *
 */

export class ListView extends LitElement {

  // internal
  static get properties() {
    return { };
  }

  // internal
  constructor() {
    super();
    this.addState = "off"
  }

  /**
   * Override to false to allow the Add button to show up.
   */
  readOnly() {
    return true;
  }

  /* onCollectionNotification.
  * Listener for model-created notifications.  This is
  * called when a new Resource has been created, and a
  * new view must be created to display that Resource.
  */

  onCollectionNotification(message, model, parameter) {
    switch(message) {
      /*
       * Create a new dw-host web component and add it as a child.
       * Because this view is a web component, adding that child component
       * queues the appropriate re-render at the correct time. Our children
       * are rendered in our <slot>.
       */
      case 'created':
        let child_view = new Resource(model);
        this.appendChild(child_view);
        break;
    }
  }
  /**
   * render
   * Render the list.  Subclasses will override but most will
   * look like the example below, the key elements being:
   *
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
}

/**
 * The SortableModelListView class is an abstract base class that is extended in
 * order to create a container widget for listing sorted kubernetes resources
 * of a single Kind. The SortableModelListView extends ModelListView, adding an
 * HTML selector for picking a "sort by" attribute.
 *
 * See ModelListView.
 *
 * To implement a SortableModelListView container element, you must extend this
 * class and override the following methods. See individual methods
 * for more details:
 *
 *   sortFn(sortByAttribute) --> must return a `compare` function, for the collection.sort()
 *     https://www.w3schools.com/js/js_array_sort.asp
 *
 *     sortFn(sortByAttribute) {
 *       return function(a, b) { return a[sortByAttribute] - b[sortByAttribute] };
 *     }
 *
 *   renderSet() --> tell us how to display the collection
 *
 */
