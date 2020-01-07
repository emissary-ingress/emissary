/*
 * map.js
 * Utility functions for Maps, since Javascript doesn't provide them.
 */

/* merge(mapA, mapB)
 * Return all key:value pairs that exist in mapA and mapB.
 *
 * The spread operator (...) converts the Map into an Array which the Map constructor then uses to initialize
 * the Map with the new key:value entries.  mapB's key:value pairs override any key:value pairs that exist in
 * mapA with the same key.
 */

export function merge(mapA, mapB) {
  return new Map(...mapA, ...mapB);
}
