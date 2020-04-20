
/**
 * CoW
 *
 * The CoW class provides a Copy on Write wrapper for arbitrary javascript objects. You can use it
 * for a number of scenarios:
 *
 *   1. Deeply copy an object:
 *
 *     let original = {
 *       ...,
 *       nested: {...},
 *       array: [..., {nested_inside_array: "eggs"}, ... ],
 *       ...
 *     }
 *
 *     let copy = new CoW(original)
 *
 *   2. Detect changes to an object:
 *
 *     console.log(CoW.changed(copy)) // -> false
 *     copy.foo = "bar"
 *     console.log(CoW.changed(copy)) // -> true
 *
 *   3. Track changes to an object:
 *
 *     console.log(CoW.deltas(copy)) // -> {foo: "bar"}
 *
 *   4. Prevent changes (create a read-only copy):
 *
 *     let original = {...}
 *     let readOnlyChangeHandler = ()=>{throw new Error("read-only")}
 *     let readOnlyCopy = new CoW(original, readOnlyChangeHandler)
 *
 *   5. Notify on change:
 *
 *     let original = {...}
 *     let notifyingChangeHandler = ()=>{console.log("original was changed!")}
 *     let copy = new CoW(original, notifyingChangeHandler)
 *
 * The CoW class is used by the implementation of the Resource model in all these scenarios, but the
 * major benefit is to automatically detect and notify listeners (i.e. views) when changes are made
 * in a Resource's underlying yaml.
 *
 * Potential caveats/future improvements:
 *
 *  - This code has been tested quite extensively for yaml-like use cases, i.e. nested objects
 *    including arrays, etc. (See ../tests/cow.js for details.) It has *not* been tested for
 *    circular references or custom classes. The design in principal should be extensible for both,
 *    however that is not currently needed.
 */
export class CoW {

  /**
   * CoW.deltas(cow)
   *
   * Return an object containing only fields that have been modified relative to the target of the
   * CoW.
   */
  static deltas(cow) {
    let target = cow[TARGET]
    let writes = cow[WRITES]

    let result = {}

    for (let prop in writes) {
      let value = writes[prop]
      if (CoW.is(value)) {
        let d = CoW.deltas(value)
        if (!deepEqual(d, {})) {
          result[prop] = d
        }
      } else {
        let orig = Reflect.get(target, prop)
        if (!deepEqual(orig, value)) {
          result[prop] = value
        }
      }
    }

    return result
  }

  /**
   * CoW.changed(cow)
   *
   * Reports whether mutation of a CoW resulted in changes.
   */
  static changed(cow) {
    return CoW.mutated(cow) && !deepEqual(CoW.deltas(cow), {})
  }

  /**
   * CoW.mutated(cow)
   *
   * Reports whether a CoW has been mutated.
   */
  static mutated(cow) {
    return cow[CHANGED]
  }

  /**
   * Detects whether an object is a CoW or not.
   */
  static is(obj) {
    return obj !== null && typeof obj === "object" && typeof obj[WRITES] === "object"
  }

  // ================== everything below here is implementation ========================

  constructor(target, onChange = noop) {
    this[TARGET] = target
    // The WRITES map holds the modified value for scalars. For nested objects and arrays it holds a
    // value even if there are no changes. That value is a CoW wrapper for objects and an ArrayProxy
    // for Arrays.
    this[WRITES] = {}
    this.onChange = onChange
    // The CHANGED flag tracks whether this object has been changed or any nested object has been
    // changed.
    this[CHANGED] = false
    return new Proxy(target, this)
  }

  trackChanges() {
    this.onChange()
    this[CHANGED] = true
  }

  // implement the get trap (see MDN Proxy documentation)
  get(obj, prop) {
    if (prop === WRITES) {
      return this[WRITES]
    }
    if (prop === CHANGED) {
      return this[CHANGED]
    }
    if (prop === TARGET) {
      return this[TARGET]
    }

    if (this[WRITES].hasOwnProperty(prop)) {
      return this[WRITES][prop]
    } else {
      let result = Reflect.get(...arguments)
      if (result !== null && typeof result === "object") {
        if (Array.isArray(result)) {
          result = new ArrayProxy(result, this.trackChanges.bind(this))
        } else {
          result = new CoW(result, this.trackChanges.bind(this))
        }
        this[WRITES][prop] = result
      }
      return result
    }
  }

  // implement the set trap (see MDN Proxy documentation)
  set(obj, prop, value) {
    this.trackChanges()
    this[WRITES][prop] = value
    return true
  }

  // implement the deleteProperty trap (see MDN Proxy documentation)
  deleteProperty(obj, prop) {
    this.trackChanges()
    this[WRITES][prop] = TOMBSTONE
    return true
  }

  // implement the ownKeys trap (see MDN Proxy documentation)
  ownKeys(obj) {
    let result = new Set(Reflect.ownKeys(...arguments))
    let writes = this[WRITES]
    for (let k of Object.keys(writes)) {
      let v = writes[k]
      if (v === TOMBSTONE) {
        result.delete(k)
      } else {
        result.add(k)
      }
    }
    result = Array.from(result)
    return result
  }

  // implement the getOwnPropertyDescriptor trap (see MDN Proxy documentation)
  getOwnPropertyDescriptor(obj, prop) {
    if (this.has(obj, prop)) {
      return { configurable: true, enumerable: true }
    }
  }

  // implement the has trap (see MDN Proxy documentation)
  has(obj, key) {
    let orig = Reflect.has(...arguments)
    if (orig) { return true }
    let writes = this[WRITES]
    return Reflect.has(writes, key)
  }

}

// we use these symbols for our own properties so that we can guarantee no collision with user defined
// property names

const TARGET = Symbol()
const WRITES = Symbol()
const CHANGED = Symbol()
const TOMBSTONE = Symbol()

const MUTATING = new Set(["reverse", "sort", "shift", "splice", "push", "pop", "unshift"])

/**
 * ArrayProxy
 *
 * A proxy for Array that invokes onChange if any mutation occurs. This is used by the CoW object to
 * handle Array properties. We don't try to be too fine grained with Arrays. If any change happens
 * to an Array, we just consider the whole array to be changed.
 */
class ArrayProxy {

  static is(obj) {
    return obj !== null && typeof obj === "object" && typeof obj[CHANGED] === "boolean"

  }

  constructor(orig, onChange) {
    this.target = orig.map((x)=>{
      if (typeof x === "object") {
        if (Array.isArray(x)) {
          return new ArrayProxy(x, this.trackChanges.bind(this))
        } else {
          return new CoW(x, this.trackChanges.bind(this))
        }
      } else {
        return x
      }
    })
    this.onChange = onChange
    this[CHANGED] = false
    return new Proxy(this.target, this)
  }

  trackChanges() {
    this.onChange()
    this[CHANGED] = true
  }

  // implement the get trap (see MDN Proxy documentation)
  get(obj, prop) {
    if (prop === CHANGED) {
      return this[CHANGED]
    }

    let value = Reflect.get(...arguments)
    if (MUTATING.has(prop)) {
      let othis = this
      return function () {
        othis.trackChanges()
        return value.bind(othis.target)(...arguments)
      }
    } else {
      return value
    }
  }

  // implement the set trap (see MDN Proxy documentation)
  set(obj, prop, value) {
    this.trackChanges()
    return Reflect.set(...arguments)
  }

}

function noop() {}

// isn't there a library or package that does this?
export function deepEqual(a, b) {
  if (a === b) {
    return true
  }

  if (a === null || b === null || typeof a !== "object" || typeof b !== "object") {
    return a === b
  }

  if (Array.isArray(a)) {
    if (Array.isArray(b) && a.length === b.length) {
      for (let i = 0; i < a.length; i++) {
        if (!deepEqual(a[i], b[i])) {
          return false
        }
      }
      return true
    }
    return false
  }

  if (Array.isArray(b)) {
    return false
  }

  // they are both non-null objects
  for (let key of Object.keys(a)) {
    if (!deepEqual(a[key], b[key])) {
      return false
    }
  }

  for (let key of Object.keys(b)) {
    if (!deepEqual(a[key], b[key])) {
      return false
    }
  }

  return true
}
