import {Model} from "../framework/model.js"

let assert = chai.assert

describe('Model', function() {
  describe('addListener', function() {
    it('should notify an added listener with the specified tag', function() {
      let m = new Model()
      let v = new MockView()
      m.addListener(v, "foo")
      m.notify()
      assert.deepEqual(v.changes, ["foo"])
    })

    it('should be able to add multiple listeners with the same tag', function() {
      let m = new Model()
      let v1 = new MockView()
      let v2 = new MockView()
      m.addListener(v1, "foo")
      m.addListener(v2, "foo")
      m.notify()
      assert.deepEqual(v1.changes, ["foo"])
      assert.deepEqual(v2.changes, ["foo"])
    })

    it('should be able to add a single listener multiple times with different tags', function() {
      let m = new Model()
      let v = new MockView()
      m.addListener(v, "foo")
      m.addListener(v, "bar")
      m.notify()
      assert.deepEqual(v.changes, ["foo", "bar"])
    })

    it('should notify a listener only once for the same tag', function() {
      let m = new Model()
      let v = new MockView()
      m.addListener(v, "foo")
      m.addListener(v, "foo")
      m.notify()
      assert.deepEqual(v.changes, ["foo"])
    })
  })

  describe('removeListener', function() {
    it('should remove the specified listener/tag combo', function() {
      let m = new Model()
      let v = new MockView()
      m.addListener(v, "foo")
      m.notify()
      assert.deepEqual(v.changes, ["foo"])
      m.removeListener(v, "foo")
      v.clear()
      m.notify()
      assert.deepEqual(v.changes, [])
    })
    it('should retain other tags of the same listener', function() {
      let m = new Model()
      let v = new MockView()
      m.addListener(v, "foo")
      m.addListener(v, "bar")
      m.notify()
      assert.deepEqual(v.changes, ["foo", "bar"])
      m.removeListener(v, "bar")
      v.clear()
      m.notify()
      assert.deepEqual(v.changes, ["foo"])
    })
    it('should retain other listeners', function() {
      let m = new Model()
      let v1 = new MockView()
      let v2 = new MockView()
      m.addListener(v1, "foo")
      m.addListener(v2, "bar")
      m.notify()
      assert.deepEqual(v1.changes, ["foo"])
      assert.deepEqual(v2.changes, ["bar"])
      m.removeListener(v2, "bar")
      v1.clear()
      v2.clear()
      m.notify()
      assert.deepEqual(v1.changes, ["foo"])
      assert.deepEqual(v2.changes, [])
    })
  })

  describe('notify', function() {
    it('should invoke listener.onModelChanged(model, tag) on all listener/tag combos when notify is called', function() {
      let m = new Model()
      let v1 = new MockView()
      let v2 = new MockView()
      m.addListener(v1, "foo")
      m.addListener(v1, "bar")
      m.addListener(v2, "moo")
      m.addListener(v2, "arf")
      m.notify()
      assert.deepEqual(v1.changes, ["foo", "bar"])
      assert.deepEqual(v2.changes, ["moo", "arf"])
    })
  })
})

class MockView {

  constructor() {
    this.changes = []
  }

  onModelChanged(model, tag) {
    this.changes.push(tag)
  }

  clear() {
    this.changes = []
  }

}
