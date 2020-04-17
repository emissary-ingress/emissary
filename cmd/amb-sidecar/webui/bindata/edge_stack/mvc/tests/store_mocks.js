import {Resource} from '../framework/resource2.js'
import {ResourceStore} from '../framework/resource_store.js'

// mock implementations for use in resource.js and resource_collection.js

var versionCounter = 0

export class MemoryStore extends ResourceStore {

  constructor(...resourceClasses) {
    super(...resourceClasses)
    this.latency = 500
  }

  apply(collection, yaml) {
    return new Promise((good, bad)=>{
      setTimeout(()=>{
        good()
        setTimeout(()=>{
          let copy = JSON.parse(JSON.stringify(yaml))
          copy.metadata.resourceVersion = (versionCounter++).toString()
          collection.load(copy)
        }, this.latency)
      })
    })
  }

  delete(collection, yaml) {
    return new Promise((good, bad)=>{
      setTimeout(()=>{
        good()
        setTimeout(()=>{
          let set = new Set()
          for (let r of collection) {
            if (!r.isNew()) {
              set.add(r.key())
            }
          }
          set.delete(Resource.yamlKey(yaml))
          collection.intersect(set)
        }, this.latency)
      })
    })
  }

}

export class SpecializedResource extends Resource {
  static get defaultYaml() {
    return {
      kind: "specialized",
      specialized: "yes"
    }
  }
}

export class GoodStore {

  new(collection, kind) {
    if (kind === "specialized") {
      return new SpecializedResource(collection)
    } else {
      return new Resource(collection)
    }
  }

  apply(collection, yaml) {
    return new Promise((good, bad)=>{
      setTimeout(()=>{
        good()
        setTimeout(()=>{
          collection.load(yaml)
        })
      })
    })
  }

  delete(collection, yaml) {
    return new Promise((good, bad)=>{
      setTimeout(()=>{
        good()
        setTimeout(()=>{
          let set = new Set(collection)
          set.delete(Resource.yamlKey(yaml))
          collection.intersect(set)
        })
      })
    })
  }

}

export class BadStore {

  new(collection, kind) {
    return new Resource(collection)
  }

  apply(collection, yaml) {
    return new Promise((good, bad)=>{
      setTimeout(()=>bad("yaml application failed!"))
    })
  }

  delete(collection, yaml) {
    return new Promise((good, bad)=>{
      setTimeout(()=>bad("delete failed!"))
    })
  }

}

// helper to generate fake yaml for testing
var count = 0
export function yamlGen() {
  return {kind: "Resource", metadata: {name: `r${count++}`, namespace: "default"}}
}
