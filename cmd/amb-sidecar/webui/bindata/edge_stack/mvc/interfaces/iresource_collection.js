/**
 * IResourceCollection
 *
 * This is the Interface class for ResourceCollection.
 */

import { ResourceCollection } from "../framework/resource_collection2.js";

export class IResourceCollection extends ResourceCollection {

  /* ====================================================================================================
   *  There are no methods that must be implemented by subclasses.
   * ====================================================================================================
   */

  /**
   * constructor(store)
   *
   * A ResourceCollection delegates loading, saving, and deleting resources to a ResourceStore.
   */
  constructor(store) {
    super(store);
  }

}
