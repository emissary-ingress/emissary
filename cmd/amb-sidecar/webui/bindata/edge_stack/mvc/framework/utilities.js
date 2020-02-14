/*
 * utilities.js
 * Utility functions for debugging, as well as for Maps, Sets, and Objects, since
 * Javascript doesn't provide them.
 */

import { hasDebugBackend } from "../../components/api-fetch.js"

/* ================================ Map Utilities ================================ */

/* mapMerge(mapA, mapB)
 * Return a Map of all key:value pairs that exist in mapA and mapB.
 *
 * The spread operator (...) converts the Map into an Array which the Map constructor then uses to initialize
 * the Map with the new key:value entries.  mapB's key:value pairs override any key:value pairs that exist in
 * mapA with the same key.
 *
 * NOTE: This is only a shallow merge, key/value pairs at the top level of the map.  It does not recursively
 * merge pairs throughout a Map tree.
 */

export function mapMerge(mapA, mapB) {
  return new Map([...mapA, ...mapB]);
}

/* mapDiff(mapA, mapB)
 * Compare two Maps that represent trees of key/value pairs.  Return a change list of entries as follows:
 * - [path, "deleted", old_value]             => At the path, the value does not exist in the new YAML.
 * - [path, "added", new_value]               => At the path, there is a new value that didn't exist in the old YAML.
 * - [path, "changed", old_value, new_value]  => At the path, a value was changed from old_value to new_value
 */

export function mapDiff(mapA, mapB) {
  throw new Error("Not yet implemented");
  return null;
}



/* ================================ Set Utilities ================================ */

/* setIsSuperset(set, subset)
 * Return true if the subset's elements all exist in the set, false otherwise.
 */

export function setIsSuperset(set, subset) {
  for (let elem of subset) {
    if (!set.has(elem)) {
      return false;
    }
  }

  return true;
}

/* setUnion(setA, setB)
 * Return a new set that has all the elements that exist in setA and setB.
 */

export function setUnion(setA, setB) {
  let _union = new Set(setA);
  for (let elem of setB) {
    _union.add(elem);
  }

  return _union;
}

/* setIntersection(setA, setB)
 * Return the elements that exist both in setA and setB.
 */

export function setIntersection(setA, setB) {
  let _intersection = new Set();
  for (let elem of setB) {
    if (setA.has(elem)) {
      _intersection.add(elem);
    }
  }

  return _intersection;
}

/* setSymmetricDifference(setA, setB)
 * Return a set that consists of the elements of setA that are not in setB,
 * and the elements in setB that are not in SetA.  This can also be computed
 * as difference(union(setA, setB), intersection(setA, setB)) or
 * the the union of the sets with the elements that are in common removed.
 */

export function setSymmetricDifference(setA, setB) {
  let _difference = new Set(setA);
  for (let elem of setB) {
    if (_difference.has(elem)) {
      _difference.delete(elem);
    }
    else {
      _difference.add(elem);
    }
  }

  return _difference;
}

/* setDifference(setA, setB)
 * Return a set that consists of the elements of setA that are not in setB.
 * Typically considered "subtraction"
 */


export function setDifference(setA, setB) {
  let _difference = new Set(setA);
  for (let elem of setB) {
    _difference.delete(elem);
  }

  return _difference;
}

/* ================================ Object Utilities ================================ */

/* objectMerge(mapA, mapB)
 * Return an Object of all key:value pairs that exist in objA and objB.  calls mapMerge.
 *
 * NOTE: This is only a shallow merge, key/value pairs at the top level of the object.  It does not recursively
 * merge pairs throughout an Object tree.
 */

export function objectMerge(objA, objB) {
  /* Convert to maps, and merge */
  let mapA   = new Map(Object.entries(objA));
  let mapB   = new Map(Object.entries(objB));
  let merge  = mapMerge(mapA, mapB);

  /* Iterate over the keys and values in the merged Map and set them in the result */
  let result = new Object();
  for (let [k, v] of merge) {
    result[k] = v;
  }

  return result;
}

/* objectDiff(objA, objB)
 * Compare two Objecvts that represent trees of key/value pairs.  Return a change list of entries as follows:
 * - [path, "deleted", old_value]             => At the path, the value does not exist in the new YAML.
 * - [path, "added", new_value]               => At the path, there is a new value that didn't exist in the old YAML.
 * - [path, "changed", old_value, new_value]  => At the path, a value was changed from old_value to new_value
 *
 * This function simply calls mapDiff.
 */

export function objectDiff(objA, objB) {
  let mapA   = new Map(Object.entries(objA));
  let mapB   = new Map(Object.entries(objB));

  return mapDiff(mapA, mapB);
}

/* objectCopy(obj)
 * Copy an object that is used for YAML/JSON data.  This copies the full tree of key/value pairs, where keys
 * and values are JSON compatible. Because an object is often used for key/value storage due to its convenient
 * literal representation, it is typically a replacement for the more appropriate Map.
 *
 * This leverages JSON to do all the work.
 * TODO: make objectCopy into a true deepCopy function.
 */

export function objectCopy(obj) {
  return JSON.parse(JSON.stringify(obj));
}

