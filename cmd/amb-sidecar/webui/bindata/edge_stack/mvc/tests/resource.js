import {ResourceCollection} from '../framework/resource_collection2.js'
import {Resource} from '../framework/resource2.js'

import {GoodStore, BadStore, yamlGen} from './store_mocks.js'

let assert = chai.assert;

describe('Resource', function() {
  describe('#save()', function() {
    it('should transition new resources to pending-create and then stored', function() {
      let c = new ResourceCollection(new GoodStore())
      let n = c.new()
      assert.equal(n.state, "new")
      let result = n.save()
      assert.equal(n.state, "pending-create")
      return result.then(()=>{
        assert.equal(n.state, "stored")
      })
    })

    it('should transition modified resources to pending-save and then stored', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      assert.equal(r.state, "stored")
      r.edit()
      assert.equal(r.state, "modified")
      r.yaml.foo = "bar"
      let result = r.save()
      assert.equal(r.state, "pending-save")
      return result.then(()=>{
        assert.equal(r.state, "stored")
      })
    })

    it('should report the error if there is a problem creating a new resource', function() {
      let c = new ResourceCollection(new BadStore())
      let n = c.new()
      assert.equal(n.state, "new")
      let result = n.save()
      assert.equal(n.state, "pending-create")
      return result.catch((e)=>{
        assert.equal(n.state, "new")
        assert.equal(e, "yaml application failed!")
      })
    })

    it('should report the error if there is a problem saving a modified resource', function() {
      let c = new ResourceCollection(new BadStore())
      let r = c.load(yamlGen())
      assert.equal(r.state, "stored")
      r.edit()
      r.yaml.foo = "bar"
      assert.equal(r.state, "modified")
      let result = r.save()
      assert.equal(r.state, "pending-save")
      return result.catch((e)=>{
        assert.equal(r.state, "modified")
        assert.equal(e, "yaml application failed!")
      })
    })

    it('should throw an error if you try to save a stored resource', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      assert.equal(r.state, "stored")
      assert.throws(()=>r.save(), /can only save new or modified or deleted/)
    })

    it('should finish saving if you save an edited resource with no yaml changes', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      assert.equal(r.state, "stored")
      r.edit()
      return r.save()
    })
  })

  describe('#cancel()', function() {
    it('should remove new resources from the collection', function() {
      let c = new ResourceCollection(new GoodStore())
      let n = c.new()
      assert.deepEqual(Array.from(c), [n])
      n.cancel()
      assert.deepEqual(Array.from(c), [])
    })

    it('should transition a modified resource back to stored', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      r.edit()
      assert.isTrue(r.isModified())
      r.cancel()
      assert.isFalse(r.isModified())
      assert.equal(r.state, "stored")
    })

    it('should throw an error if you cancel a non-new, non-modified, non-deleted', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      assert.throws(()=>r.cancel(), /cannot cancel/)
    })
  })

  describe('#delete()', function() {
    it('should transition a stored resource to deleted', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      assert.equal(r.state, "stored")
      r.delete()
      assert.equal(r.state, "deleted")
    })
    it('should transition a resource to pending-deleted on save', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      assert.equal(r.state, "stored")
      r.delete()
      assert.equal(r.state, "deleted")
      r.save()
      assert.equal(r.state, "pending-delete")
    })
    it('should remove resource from collection after successful save', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      assert.equal(r.state, "stored")
      r.delete()
      assert.equal(r.state, "deleted")
      let result = r.save().then(()=>{
        assert.isFalse(c.contains(r))
      })
      assert.equal(r.state, "pending-delete")
      return result
    })
    it('should report errors encounterd during save and transition back to deleted state', function() {
      let c = new ResourceCollection(new BadStore())
      let r = c.load(yamlGen())
      assert.equal(r.state, "stored")
      r.delete()
      assert.equal(r.state, "deleted")
      let result = r.save().catch((e)=>{
        assert.equal(e, "delete failed!")
        assert.isTrue(c.contains(r))
        assert.equal(r.state, "deleted")
      })
      assert.equal(r.state, "pending-delete")
      return result
    })
  })

  describe('#load(yaml)', function() {
    it('should transition resources from pending-create to stored', function() {
      let c = new ResourceCollection(new GoodStore())
      let n = c.new()
      assert.equal(n.state, "new")
      n.save()
      assert.equal(n.state, "pending-create")
      n.load(n.yaml)
      assert.equal(n.state, "stored")
    })
  })

  describe('#isNew()', function() {
    it('should return true for new resources', function() {
      let c = new ResourceCollection(new GoodStore())
      let n = c.new()
      assert.isTrue(n.isNew())
    })
    it('should return false for stored resources', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      assert.isFalse(r.isNew())
    })
    it('should transition from true to false when a new resource is saved', function() {
      let c = new ResourceCollection(new GoodStore())
      let n = c.new()
      assert.isTrue(n.isNew())
      return n.save().then(()=>{
        assert.isFalse(n.isNew())
      })
    })
  })

  describe('#isModified()', function() {
    it('should return true for edited resources', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      r.edit()
      assert.isTrue(r.isModified())
    })
    it('should return false for unedited resources', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      assert.isFalse(r.isModified())
    })
    it('should return false for new resources', function() {
      let c = new ResourceCollection(new GoodStore())
      let n = c.new()
      assert.isFalse(n.isModified())
    })
    it('should transition from true to false when an edited resource is saved', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      assert.isFalse(r.isModified())
      r.edit()
      r.yaml.foo = "bar"
      assert.isTrue(r.isModified())
      return r.save().then(()=>{
        assert.isFalse(r.isModified())
      })
    })
  })

  describe('#isDeleted()', function() {
    it('should return false for new resources', function() {
      let c = new ResourceCollection(new GoodStore())
      let n = c.new()
      assert.isFalse(n.isDeleted())
    })
    it('should return false for stored resources', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      assert.isFalse(r.isDeleted())
    })
    it('should return false for edited resources', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      r.edit()
      assert.isFalse(r.isDeleted())
    })
    it('should return true for saved resources after .delete() is called', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      r.delete()
      assert.isTrue(r.isDeleted())
    })
  })

  describe('#isReadOnly()', function() {
    it('should be false for new resources', function() {
      let c = new ResourceCollection(new GoodStore())
      let n = c.new()
      assert.isFalse(n.isReadOnly())
    })
    it('should be true for stored resources', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      assert.isTrue(r.isReadOnly())
    })
    it('should be false for an edited resource', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      r.edit()
      assert.isFalse(r.isReadOnly())
    })
  })

  describe('#edit()', function() {
    it('should allow edits of stored resources', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      assert.throws(()=>{r.yaml.foo = "bar"}, /read-only/)
      r.edit()
      r.yaml.foo = "bar"
      assert.equal(r.yaml.foo, "bar")
    })
    it('should throw an error for deleted resources', function() {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      r.delete()
      assert.throws(()=>r.edit(), /cannot edit a deleted resource/)
    })
  })

  describe('#yaml', function() {
    it('should notify when you modify yaml', function(done) {
      let c = new ResourceCollection(new GoodStore())
      let r = c.load(yamlGen())
      let calledDone = false
      r.addListener(new ChangeListener(()=>{
        if (!calledDone) {
          done()
          calledDone = true
        }
      }))
      r.edit()
      r.yaml.foo = "bar"
    })
  })
})

export class ChangeListener {
  constructor(action) {
    this.action = action
  }
  onModelChanged(model, tag, message) {
    this.action(message)
  }
}
