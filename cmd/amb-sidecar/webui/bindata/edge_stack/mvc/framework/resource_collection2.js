import {Model} from "./model2.js"
import {Resource} from "./resource2.js"

/**
 * The Resource and ResourceCollection classes provide specialized implementations of Model that
 * implement identity for kubernetes resources, along with the state machine associated with
 * persisting creation, edits, and deletes of resources to a remote store.
 *
 * The details of the remote store are factored into a separate ResourceStore class in order to
 * allow for multiple store implementations.
 *
 * A Resource and its ResourceCollection are tightly coupled. The ResourceCollection ensures that
 * there is only ever a single resource for a given identity. The ResourceCollection also helps
 * track the state involved in editing and saving each resource.
 *
 * A Resource cannot exist without a ResourceCollection. Every Resource has a reference to its
 * containing ResourceCollection. Resources are never constructed directly, but instead constructed
 * via the new() and load() methods on the ResourceCollection.
 *
 * A Resource can be in the following states:
 *
 * - nonexistent       (not an explicitly represented state, but useful for the diagram)
 * - new               (exists in memory, but not stored)    
 *   + pending-create  (exists in memory, awaiting confirmation of creation in storage)
 * - modified          (exists in memory and storage, but with updates in memory)
 *   + pending-save    (exists in memory, awaiting confirmation of updates being stored)
 *   + conflicted      (exists in memory, cannot be stored due to simultaneous edit attempts aka mid-air collision)
 *   + zombie          (exists in memory with updates, deleted from storage)
 * - stored            (exists in memory and storage with no in-memory or pending changes)
 * - deleted           (exists in storage, but "soft deleted" in memory)
 *   + pending-delete  (soft deleted in memory, awaiting confirmation of deletion in storage)
 *
 *
 * The diagram below illustrates the state transitions. The key for transition labels is:
 *
 *
 *               Add/C.new()       : The "Add" button, or ResoureCollection.new()
 *               Edit/R.edit()     : The "Edit" button or Resource.edit()
 *               Save/R.save()     : The "Save" button or Resource.save()
 *               Delete/R.delete() : The "Delete" button or Resource.delete()
 *               Store RT          : A round-trip to the persistent store.
 *               Store UP          : An async update from the persistent store.
 *
 *
 *                   Add/C.new()                                 Store RT
 *        +-------------------------------nonexistent<--------------------------------+
 *        |                                                                           |
 *        |                                                                           |
 *        |                                                                           |
 *        |                                                                     pending-delete
 *        |                                                                          /|\
 *        |                                                                           |
 *        |                                                                           | R.save()
 *       \|/  Save/R.save()                  Store RT           Delete/R.delete()     |
 *       new --------------->pending-create------------>stored-------------------->deleted
 *                                                       | /|\
 *                                                       |  |
 *                                        Edit/R.edit()  |  |     Store RT
 *                                             +---------+  +-------------+
 *                                             |                          |
 *                                             |                          |
 *                               Store UP     \|/                         |
 *                      zombie<------------modified----------------->pending-save
 *                                             |     Save/R.save()        |
 *                                             |                          |
 *                                             |                          |
 *                                    Store UP |                          | Store RT
 *                                             |                          |
 *                                             +------->conflicted<-------+
 *
 */
export class ResourceCollection extends Model {

  constructor(store) {
    super()
    this.store = store
    this.new_resources = new Set();
    this.resources = new Map();
  }

  new(kind) {
    let result = this.store.new(this, kind)
    this.new_resources.add(result)
    result.yaml = result.constructor.defaultYaml
    this.notify()
    return result
  }

  load(yaml) {
    let key = Resource.yamlKey(yaml)
    var resource
    if (this.resources.has(key)) {
      resource = this.resources.get(key)
    } else {
      let kind = yaml.kind
      resource = this.store.new(this, kind)
      this.resources.set(key, resource)
      this.notify()
    }
    resource.load(yaml)
    return resource
  }

  intersect(keys) {
    let notify = false
    for (let [k, r] of this.resources.entries()) {
      if (!keys.has(k) && !r.isNew()) {
        notify = true
        if (r.isModified()) {
          r.storedYaml = null
          r.storedVersion = null
          r.notify()
        } else {
          this.resources.delete(k)
          if (r.isDeleted()) {
            r.resolvePending()
          }
        }
      }
    }
    if (notify) {
      this.notify()
    }
  }

  reconcile(yamls) {
    let keys = new Set()
    for (let yaml of yamls) {
      let r = this.load(yaml)
      keys.add(r.key())
    }
    this.intersect(keys)
  }

  *[Symbol.iterator]() {
    // yield all the new resources first
    for (let n of this.new_resources) {
      yield n
    }

    // yield all the stored resources (some of which may be modified)
    for (let r of this.resources.values()) {
      yield r
    }
  }

  contains(r) {
    return this.new_resources.has(r) || this.resources.has(r.key())
  }

}
