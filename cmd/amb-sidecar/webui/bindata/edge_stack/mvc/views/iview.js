/**
 * View
 * This is the interface class for Views.
 *
 */

import { View } from "./view.js"

export class IView extends View {

  /* properties
   * These are the properties of the View, which reflect the properties of the underlying Model,
   * and also include transient state (e.g. viewState).
   *
   * LitElement manages these declared properties and provides various services depending on how they are used.
   * https://lit-element.polymer-project.org/guide/properties
   */

  static get properties() {
    return {
      viewState: {type: String},  // View
    }
  }

  /* ====================================================================================================
   *  These methods must be implemented by subclasses.
   * ====================================================================================================
   */

  /* constructor
   * The View constructor, which takes a Model (model) as its parameter.
   */

  constructor(model) {
    super(model);
  }

  /**
   * This method is called on the View when the View needs to match the current state of its Model.
   * Generally this happens during initialization and during editing when the Cancel button is pressed and the
   * View reverts to displaying the original Model's state.
   */
  readFromModel() {
    throw new Error("please implement View:readFromModel()")
  }

  /**
   * This method is called on the View when the View has new, validated state that should be written back
   * to the Model.  This happens during a Save operation after the user has modified the View.
   */

  writeToModel() {
    throw new Error("please implement View:writeToModel()")
  }


  renderSelf() {
    throw new Error("please implement View:renderSelf()")
  }

  /* ====================================================================================================
   *  Subclasses do not implement these methods.  They are implemented by View and may be used by
   *  subclasses directly.
   * ====================================================================================================
   */
}
