import {Snapshot} from "../../components/snapshot.js";
import {getCookie} from "../../components/cookies.js";
import { ApiFetch } from "../../components/api-fetch.js";

export class ResourceStore {

  constructor(...resourceClasses) {
    this.kinds = new Map()
    for (let cls of resourceClasses) {
      let yaml = cls.defaultYaml
      let kind = yaml.kind
      if (typeof kind !== "string") {
        throw new Error("resource classes must define a \"static get defaultYaml() {...}\" property that includes the kind field")
      }
      this.kinds.set(kind, cls)

      // set the default kind to the first class
      if (this.defaultKind === undefined) {
        this.defaultKind = kind
      }
    }

    if (this.kinds.size === 0) {
      throw new Error("a Store requires at least one resource class to be constructed")
    }
  }

  new(collection, kind) {
    kind = kind || this.defaultKind
    if (!this.kinds.has(kind)) {
      throw new Error("unrecognized kind: " + kind)
    }
    let cls = this.kinds.get(kind)
    return new cls(collection)
  }

  subscribe(collection) {
    Snapshot.subscribe((snapshot)=>{
      let resources = []
      for (let kind of this.kinds.keys()) {
        resources.push(...snapshot.getResources(kind))
      }
      collection.reconcile(resources)
    })
  }

  apply(collection, yaml) {
    let copy = JSON.parse(JSON.stringify(yaml))
    this.strip(copy)
    console.log(copy)
    let cookie = getCookie("edge_stack_auth")
    let params = {
      method: "POST",
      headers: new Headers({'Authorization': 'Bearer ' + cookie}),
      body: JSON.stringify(copy)
    }

    /* Make the call to apply */
    return new Promise((good, bad)=>{
      ApiFetch('/edge_stack/api/apply', params)
        .then(r => {
          r.text().then(t => {
            if (r.ok) {
              good()
            } else {
              bad(t)
            }
          })
        })
    })
  }

  strip(yaml) {
    delete yaml.status
    let metadata = yaml.metadata || {}
    delete metadata.uid
    delete metadata.selfLink
    delete metadata.generation
    delete metadata.resourceVersion
    delete metadata.creationTimestamp
    let annotations = metadata.annotations || {}
    delete annotations["kubectl.kubernetes.io/last-applied-configuration"]
  }

  delete(collection, yaml) {
    let cookie = getCookie("edge_stack_auth");
    let params = {
      method: "POST",
      headers: new Headers({ 'Authorization': 'Bearer ' + cookie }),
      body: JSON.stringify({
        Namespace: yaml.metadata.namespace,
        Names: [`${yaml.kind}/${yaml.metadata.name}`]
      })
    };

    return new Promise((good, bad)=>{
      ApiFetch('/edge_stack/api/delete', params).then(
        r=>{
          r.text().then(t=>{
            if (r.ok) {
              good()
            } else {
              bad(`Unexpected error while deleting resource: ${r.statusText}`)
            }
          })
        })
    })
  }

}
