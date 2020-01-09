/*
 * map.js
 * Utility functions for Maps, since Javascript doesn't provide them.
 */

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
