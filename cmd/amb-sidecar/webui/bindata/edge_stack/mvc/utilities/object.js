/*
 * object.js
 * Utility functions for Objects, since Javascript doesn't provide them.
 */

import { mapMerge } from "./map.js"

/* objectMerge(mapA, mapB)
 * Return a new object that is the merge of all key:value pairs that exist in objA and objB.
 * We first convert the objects into Maps, merge them, and then return a new object from the key/value pairs
 * in the resulting Map.  This is useful for situations where, since Maps have no literal representation,
 * the key/value pairs are listed in { key: value, key2: value2, ... } form which constructs an Object.
 * This allows Objects to be used as Maps and merged in the same way.
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
