/*
 * set.js
 * Utility functions for sets, since Javascript doesn't provide them.
 */

/* isSuperset(set, subset)
 * Return true if the subset's elements all exist in the set, false otherwise.
 */

export function isSuperset(set, subset) {
  for (var elem of subset) {
    if (!set.has(elem)) {
      return false;
    }
  }

  return true;
}

/* union(setA, setB)
 * Return a new set that has all the elements that exist in setA and setB.
 */

export function union(setA, setB) {
  var _union = new Set(setA);
  for (var elem of setB) {
    _union.add(elem);
  }

  return _union;
}

/* intersection(setA, setB)
 * Return the elements that exist both in setA and setB.
 */

export function intersection(setA, setB) {
  var _intersection = new Set();
  for (var elem of setB) {
    if (setA.has(elem)) {
      _intersection.add(elem);
    }
  }

  return _intersection;
}

/* symmetricDifference(setA, setB)
 * Return a set that consists of the elements of setA that are not in setB,
 * and the elements in setB that are not in SetA.  This can also be computed
 * as difference(union(setA, setB), intersection(setA, setB)) or
 * the the union of the sets with the elements that are in common removed.
 */

export function symmetricDifference(setA, setB) {
  var _difference = new Set(setA);
  for (var elem of setB) {
    if (_difference.has(elem)) {
      _difference.delete(elem);
    }
    else {
      _difference.add(elem);
    }
  }

  return _difference;
}

/* difference(setA, setB)
 * Return a set that consists of the elements of setA that are not in setB.
 * Typically considered "subtraction"
 */


export function difference(setA, setB) {
  var _difference = new Set(setA);
  for (var elem of setB) {
    _difference.delete(elem);
  }

  return _difference;
}
