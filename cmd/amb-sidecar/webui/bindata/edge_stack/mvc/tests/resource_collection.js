import {ResourceCollection} from '../framework/resource_collection.js'
import {Resource} from '../framework/resource.js'

import {GoodStore, SpecializedResource, yamlGen} from './store_mocks.js'

let assert = chai.assert;

describe('ResourceCollection', function() {

  describe("#new()", function() {
    it('should return a new Resource', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.new()
      assert(r.isNew())
    })
    it('should return a non-pending Resource', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.new()
      assert(!r.isPending())
    })
    it('should return a unique Resource', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.new()
      assert(c.new() !== r)
    })
    it('should be contained in the collection', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.new()
      assert(c.contains(r))
    })
    it('should be returned by the iterator', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.new()
      assert(new Set(c).has(r))
    })
    it('should allow resource specialization', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.new("specialized")
      assert(r instanceof SpecializedResource)
    })
    it('should return a resource with a unique key', function() {
      let c = new ResourceCollection(new GoodStore())
      let n1 = c.new()
      let n2 = c.new()
      assert.notEqual(n1.key(), n2.key())
    })
    it('should return a resource whose unique key solidifies on save', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.new()
      let y = {
        kind: "kind",
        metadata: {
          name: "bob",
          namespace: "space"
        }
      }
      r.yaml = y
      return r.save().then(()=>{
        assert.equal(r.key(), Resource.yamlKey(y))
      })
    })
    it('should return a resource with defaulted yaml', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.new("specialized")
      assert.equal(r.yaml.specialized, "yes")
    })
  })

  describe("#load(yaml)", function() {
    it('should return a non-modified resource', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load({kind: "Resource", metadata: {name: "foo", namespace: "bar"}})
      assert(!r.isModified())
    })
    it('should return a read-only resource', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load({kind: "Resource", metadata: {name: "foo", namespace: "bar"}})
      assert(r.isReadOnly())
    })
    it('should return a read-only resource that does not allow writes', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load({kind: "Resource", metadata: {name: "foo", namespace: "bar"}})
      assert.throws(()=>{r.yaml.foo = "bar"}, /read-only/)
    })
    it('should return a resource that is contained in the collection', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load({kind: "Resource", metadata: {name: "foo", namespace: "bar"}})
      assert(c.contains(r))
    })
    it('should return a resource that is returned by the iterator', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load({kind: "Resource", metadata: {name: "foo", namespace: "bar"}})
      assert(new Set(c).has(r))
    })
    it('should instantiate specialized resources', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load({kind: "specialized", metadata: {name: "foo", namespace: "bar"}})
      assert(r instanceof SpecializedResource)
    })
  })

  describe("@@iterator()", function() {
    it('should return new resources', function() {
      let c = new ResourceCollection(new GoodStore())
      let n1 = c.new()
      assert.deepEqual(Array.from(c), [n1])
    })
    it('should return every new resource in order', function() {
      let c = new ResourceCollection(new GoodStore())
      let n1 = c.new()
      let n2 = c.new()
      assert.deepEqual(Array.from(c), [n1, n2])
    })
    it('should return loaded resources', function() {
      let c = new ResourceCollection(new GoodStore())
      let n1 = c.new()
      let n2 = c.new()
      let r1 = c.load(yamlGen())
      assert.deepEqual(Array.from(c), [n1, n2, r1])
    })
    it('should return every loaded resource in order', function() {
      let c = new ResourceCollection(new GoodStore())
      let n1 = c.new()
      let n2 = c.new()
      let r1 = c.load(yamlGen())
      let r2 = c.load(yamlGen())
      assert.deepEqual(Array.from(c), [n1, n2, r1, r2])
    })
    it('should return modified resources', function() {
      let c = new ResourceCollection(new GoodStore())
      let n1 = c.new()
      let n2 = c.new()
      let r1 = c.load(yamlGen())
      let r2 = c.load(yamlGen())
      r2.edit()
      assert.deepEqual(Array.from(c), [n1, n2, r1, r2])
    })
    it('should return modified resources in storage order', function() {
      let c = new ResourceCollection(new GoodStore())
      let n1 = c.new()
      let n2 = c.new()
      let r1 = c.load(yamlGen())
      let r2 = c.load(yamlGen())
      r2.edit()
      r1.edit()
      assert.deepEqual(Array.from(c), [n1, n2, r1, r2])
    })
    it('should return modified resources even though they have been deleted from storage', function() {
      let c = new ResourceCollection(new GoodStore())
      let n1 = c.new()
      let n2 = c.new()
      let r1 = c.load(yamlGen())
      let r2 = c.load(yamlGen())
      r2.edit()
      r1.edit()

      c.intersect(new Set())
      assert.sameMembers(Array.from(c), [n1, n2, r1, r2])
    })
    it('should return modified resources in storage order even when they have been deleted from storage', function() {
      let c = new ResourceCollection(new GoodStore())
      let n1 = c.new()
      let n2 = c.new()
      let r1 = c.load(yamlGen())
      let r2 = c.load(yamlGen())
      r2.edit()
      r1.edit()

      c.intersect(new Set())
      assert.deepEqual(Array.from(c), [n1, n2, r1, r2])
    })
    it('should return deleted resources that are pending', function() {
      let c = new ResourceCollection(new GoodStore())
      let n1 = c.new()
      let n2 = c.new()
      let r1 = c.load(yamlGen())
      let r2 = c.load(yamlGen())
      r2.delete()
      r2.save()
      assert.isTrue(r2.isPending())
      assert.equal(r2.state, "pending-delete")
      assert.deepEqual(Array.from(c), [n1, n2, r1, r2])
    })
  })

  describe('#intersect(keys)', function() {
    it('should keep only resources that match keys', function() {
      let c = new ResourceCollection(new GoodStore())
      let r1 = c.load(yamlGen())
      let r2 = c.load(yamlGen())
      c.intersect(new Set([r1.key()]))
      assert.deepEqual(Array.from(c), [r1])
    })
  })

  describe('#reconcile(yamls)', function() {
    it('should load all yamls and then intersect the keys', function() {
      let yamls = [yamlGen(), yamlGen(),  yamlGen(), yamlGen()]
      let c = new ResourceCollection(new GoodStore())

      function recssert(yamls) {
        c.reconcile(yamls)
        let collectionKeys = Array.from(c).map(r=>r.key())
        let yamlKeys = yamls.map(y=>Resource.yamlKey(y))
        assert.sameMembers(collectionKeys, yamlKeys)
      }

      recssert(yamls)
      recssert(yamls.slice(0, -1))
      recssert(yamls.slice(0, -2))
      recssert(yamls.slice(1))
      recssert(yamls.slice(2))
      recssert(yamls)
    })
  })
})
