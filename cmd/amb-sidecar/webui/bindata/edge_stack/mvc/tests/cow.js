import {CoW} from '../framework/cow.js'

let assert = chai.assert;

describe('CoW', function() {
  it('should pass through gets to the target', function() {
    let target = {
      pie: "blueberry",
      pi: 3.14,
      pies: ["apple", "banana-cream", "coconut-cream"],
      ice: {
        pie: "frozen-custard",
        pi: 3.1416,
        pies: ["ice-cream", "frozen-custard"],
        cream: {
          pie: "vanilla",
          pi: 3.1416,
          pies: ["coffee", "oreo", "mint-choc-chip"]
        }
      }
    }
    let cow = new CoW(target)
    assert.deepEqual(cow, target)
  })

  it('should recursively intercept sets', function() {
    let target = {
      pie: "blueberry",
      pi: 3.14,
      pies: ["apple", "banana-cream", "coconut-cream"],
      ice: {
        pie: "frozen-custard",
        pi: 3.1416,
        pies: ["ice-cream", "frozen-custard"],
        cream: {
          pie: "vanilla",
          pi: 3.1416,
          pies: ["coffee", "oreo", "mint-choc-chip"]
        }
      }
    }
    let copy = JSON.parse(JSON.stringify(target))
    let cow = new CoW(target)
    cow.pie = "apple"
    assert.deepEqual(copy, target)
    cow.ice.cream.pie = "apple"
    assert.deepEqual(copy, target)
    assert.equal(cow.pie, "apple")
    assert.equal(cow.ice.cream.pie, "apple")
  })

  it('should allow access to deltas', function() {
    let target = {
      pie: "blueberry",
      pi: 3.14,
      pies: ["apple", "banana-cream", "coconut-cream"],
      ice: {
        pie: "frozen-custard",
        pi: 3.1416,
        pies: ["ice-cream", "frozen-custard"],
        cream: {
          pie: "vanilla",
          pi: 3.1416,
          pies: ["coffee", "oreo", "mint-choc-chip"]
        }
      }
    }

    let cow = new CoW(target)
    cow.pie = "apple"
    cow.ice.cream.pie = "apple"
    assert.deepEqual(CoW.deltas(cow), {pie: "apple", ice: {cream: {pie: "apple"}}})
  })

  it('should not mutate arrays', function() {
    let target = {
      pi: 3.14,
      pies: ["apple", "banana-cream", "coconut-cream"]
    }
    let copy = JSON.parse(JSON.stringify(target))
    let cow = new CoW(target)
    cow.pies.push("dulce-de-lece")
    assert.deepEqual(copy, target)
    assert.equal(cow.pies[cow.pies.length-1], "dulce-de-lece")
    cow.pies[1] = "banana-foster"
    assert.deepEqual(copy, target)
    assert.equal(cow.pies[1], "banana-foster")
  })

  it('should proxy objects nested in arrays', function() {
    let target = {
      pi: 3.14,
      pies: ["apple", {banana: "cream"}, "coconut-cream"]
    }
    let copy = JSON.parse(JSON.stringify(target))
    let cow = new CoW(target)
    cow.pies[1].banana = "custard"
    assert.deepEqual(copy, target)
    assert.equal(cow.pies[1].banana, "custard")
  })

  it('should notify on change', function(done) {
    let target = {
      pi: 3.14,
      pies: ["apple", {banana: "cream"}, "coconut-cream"]
    }
    let cow = new CoW(target, ()=>{
      done()
    })
    cow.pi = 3.14159
  })

  it('should notify on change of nested objects', function(done) {
    let target = {
      pi: 3.14,
      pies: ["apple", {banana: "cream"}, "coconut-cream"]
    }
    let cow = new CoW(target, ()=>{
      done()
    })
    cow.pies[1].banana = "foster"
  })

  it('should notify on change to arrays', function(done) {
    let target = {
      pi: 3.14,
      pies: ["apple", {banana: "cream"}, "coconut-cream"]
    }
    let cow = new CoW(target, ()=>{
      done()
    })
    cow.pies[1] = "banana-cream"
  })

  it('should only include changed arrays in deltas', function() {
    let target = {
      pi: 3.14,
      pies: ["apple", {banana: "cream"}, "coconut-cream"]
    }
    let cow = new CoW(target)
    cow.pi = 5
    cow.pies[1] = "banana-cream"
    assert.deepEqual(CoW.deltas(cow), {
      pi: 5,
      pies: ["apple", "banana-cream", "coconut-cream"]
    })
  })

  it('should only include changed arrays in deltas', function() {
    let target = {
      pi: 3.14,
      pies: ["apple", {banana: "cream"}, "coconut-cream"]
    }
    let cow = new CoW(target)
    cow.pi = 5
    cow.pies[1].banana = "foster"
    assert.deepEqual(CoW.deltas(cow), {
      pi: 5,
      pies: ["apple", {banana: "foster"}, "coconut-cream"]
    })
  })

  it('should track changes and mutations', function() {
    let target = {}
    let cow = new CoW(target)
    assert.isFalse(CoW.mutated(cow))
    assert.isFalse(CoW.changed(cow))
    cow.foo = "bar"
    assert.isTrue(CoW.mutated(cow))
    assert.isTrue(CoW.changed(cow))
  })

  it('should distinguish between changes and mutations', function() {
    let target = {foo: "bar"}
    let cow = new CoW(target)
    assert.isFalse(CoW.mutated(cow))
    assert.isFalse(CoW.changed(cow))
    cow.foo = "bar"
    assert.isTrue(CoW.mutated(cow))
    assert.isFalse(CoW.changed(cow))
  })

  it('should track changes and mutations in nested objects', function() {
    let target = {
      foo: {bar: "baz"}
    }
    let cow = new CoW(target)
    assert.isFalse(CoW.mutated(cow))
    assert.isFalse(CoW.changed(cow))
    cow.foo.bar = "moo"
    assert.isTrue(CoW.mutated(cow))
    assert.isTrue(CoW.changed(cow))
  })

  it('should distinguish between changes and mutations in nested objects', function() {
    let target = {
      foo: {bar: "moo"}
    }
    let cow = new CoW(target)
    assert.isFalse(CoW.mutated(cow))
    assert.isFalse(CoW.changed(cow))
    cow.foo.bar = "moo"
    assert.isTrue(CoW.mutated(cow))
    assert.isFalse(CoW.changed(cow))
  })

  it('should track changes and mutations in nested objects underneath arrays', function() {
    let target = {
      foo: ["blah", {bar: "baz"}, "blah"]
    }
    let cow = new CoW(target)
    assert.isFalse(CoW.mutated(cow))
    assert.isFalse(CoW.changed(cow))
    cow.foo[1].bar = "moo"
    assert.isTrue(CoW.mutated(cow))
    assert.isTrue(CoW.changed(cow))
  })

  it('should distinguish between changes and mutations in nested objects underneath arrays', function() {
    let target = {
      foo: ["blah", {bar: "moo"}, "blah"]
    }
    let cow = new CoW(target)
    assert.isFalse(CoW.mutated(cow))
    assert.isFalse(CoW.changed(cow))
    cow.foo[1].bar = "moo"
    assert.isTrue(CoW.mutated(cow))
    assert.isFalse(CoW.changed(cow))
  })

  it('should stringify', function() {
    let target = {key: "value"}
    let cow = new CoW(target)
    cow.foo = "bar"
    assert.equal(JSON.stringify(cow), JSON.stringify({key: "value", foo: "bar"}))
  })

  it('should report new keys', function() {
    let target = {key: "value"}
    let cow = new CoW(target)
    cow.foo = "bar"
    assert.sameMembers(Object.keys(cow), ["key", "foo"])
  })

  it('should track deletes', function() {
    let target = {key: "value"}
    let copy = JSON.parse(JSON.stringify(target))
    let cow = new CoW(target)
    assert.isFalse(CoW.changed(cow))
    assert.sameMembers(Object.keys(cow), ["key"])
    delete cow.key
    assert.isTrue(CoW.changed(cow))
    assert.sameMembers(Object.keys(cow), [])
    assert.deepEqual(target, copy)
  })

  it('should deal with null values', function() {
    let target = {key: null}
    let cow = new CoW(target)
    assert.deepEqual(cow, target)
  })

  it('should have no deltas when writing a new value that is the same as the old', function() {
    let values = [null, undefined, "pie", 3.14159, [1, 2, 3]]
    let sames = [null, undefined, "pie", 3.14159, [1, 2, 3]]
    let diffs = [undefined, null, "pi", 6.28, [3, 2, 1]]
    for (let i = 0; i < values.length; i++) {
      let value = values[i]
      let same = sames[i]
      let diff = diffs[i]

      let target = {key: value}
      let cow = new CoW(target)
      cow.key = same
      assert.deepEqual(CoW.deltas(cow), {})
      cow.key = diff
      assert.deepEqual(CoW.deltas(cow), {key: diff})
    }
  })
})
