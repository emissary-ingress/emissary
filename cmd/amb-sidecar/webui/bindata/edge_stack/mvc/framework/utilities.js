/*
 * utilities.js
 * Utility functions for debugging, as well as for Maps, Sets, and Objects, since
 * Javascript doesn't provide them.
 */

import { hasDebugBackend } from "../../components/api-fetch.js"

/* ================================ Debugging  ================================ */

export function enableMVC() {
  return true;
  /* return hasDebugBackend */
}

/* ================================ Map Utilities ================================ */

/* mapMerge(mapA, mapB)
 * Return a Map of all key:value pairs that exist in mapA and mapB.
 *
 * The spread operator (...) converts the Map into an Array which the Map constructor then uses to initialize
 * the Map with the new key:value entries.  mapB's key:value pairs override any key:value pairs that exist in
 * mapA with the same key.
 */

export function mapMerge(mapA, mapB) {
  return new Map([...mapA, ...mapB]);
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
