import {LitElement} from '../../vendor/lit-element.min.js'
import {Model} from './model2.js'

export {html, css, repeat} from '../../vendor/lit-element.min.js'

/**
 * The View class provides a base class for building web-components
 * that are Views of models. You can use it create web-components,
 * just like you would use LitElement. The only difference is that if
 * you put an instance of a Model class in a declared property, then
 * this class will automatically register itself as a listener and be
 * updated whenever that Model changes:
 *
 *   class MyModel extends Model {}
 *
 *   class MyView extends View {
 *
 *     static get properties() {
 *       return {
 *         model: {type: MyModel}
 *       }
 *     }
 *
 *     constructor() {
 *       super()
 *       this.model = null
 *     }
 *
 *     render() {
 *       return html`<p>${this.model.importantState()}</p>`
 *     }
 *   } 
 *
 *   customElements.define("my-view", MyView)
 *
 * You can have any number of "Model subclass" properties as you like
 * and they can be named whatever you like.
 *
 * For more info on lit-element and lit-html, please read the following:
 *   - https://lit-element.polymer-project.org/guide
 *   - https://lit-html.polymer-project.org/guide
 */
export class View extends LitElement {

  // The LitElement.createProperty static method is intended to be
  // overridden in order to customize how property getters/setters are
  // automatically created based on property descriptors. This
  // implementation is identical to the default implementation in
  // LitElement with the exception that if a value is an instance of
  // the Model class, then the setter will automatically
  // register/unregister the web-component as a listener.
  static createProperty(name, options) {
    this._ensureClassProperties()
    this._classProperties.set(name, options)
    if (options.noAccessor || this.prototype.hasOwnProperty(name)) {
      return
    }
    const key = typeof name === 'symbol' ? Symbol() : `__${name}`
    Object.defineProperty(this.prototype, name, {
      get() {
        return this[key]
      },
      set(value) {
        const oldValue = this[name]
        if (oldValue instanceof Model) {
          oldValue.removeListener(this, name)
        }
        if (value instanceof Model) {
          value.addListener(this, name)
        }
        this[key] = value
        this.requestUpdate(name)
      },
      configurable: true,
      enumerable: true
    })
  }

  onModelChanged(model, property) {
    this.requestUpdate(property)
  }

}
