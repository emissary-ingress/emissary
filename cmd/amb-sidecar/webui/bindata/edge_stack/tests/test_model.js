/**
 * Test_Model
 * Instantiate this class with a model to test the Model's functionality, or call Test_Model.run() to run
 * a standard suite.
 */

import { Model } from "../models/model.js"

export class Test_Model {

  /* constructor()
   * Here the model initializes any internal state including any structures for storing Listeners
   * that have subscribed to the Model.
   */

  constructor(model) {
    this.model = model;
  }

  static run() {
    let model     = new Model();
    let testSuite = new Test_Model(model);
    /* ... */
  }

  test_add_listener() {
    /* ... */
  }

  test_remove_listener() {
    /* ... */
  }

  test_notify_listeners() {
    /* ... */
  }
}

