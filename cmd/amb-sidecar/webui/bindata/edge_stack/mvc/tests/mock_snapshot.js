/**
 * MockSnapshot
 * Instantiate this class with a file path to mock snapshot data.  Running this
 */

import { Model } from "../models/model.js"

export class MockSnapshot {

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

