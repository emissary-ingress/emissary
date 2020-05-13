/**
 * ProjectCollection
 * This is an IResourceCollection subclass for Project objects.
 */

import { ProjectResource } from "./project_resource.js"
import { IResourceCollection } from "../interfaces/iresource_collection.js";
import { ResourceStore } from "../framework/resource_store.js";
import { getCookie } from '../../components/cookies.js';
import { ApiFetch } from "../../components/api-fetch.js";
import { deepEqual } from '../framework/cow.js';

const Second = 1000;

class ProjectStore extends ResourceStore {

  subscribe(collection) {
    this.delay = 1*Second
    let looper = ()=>{
      this._poll(collection);
      // This makes sure we give up if we get logged out. Out of an
      // abundance of caution we retry a few times before giving up
      // just in case this code runs super early in the login process
      // before the edge_stack_auth cookie is actually set.
      if (this.delay < 4*Second) {
        setTimeout(looper, this.delay);
      }
    }
    looper();
  }

  _poll(collection) {
    ApiFetch(`/edge_stack/api/projects/kale-snapshot`, {
      headers: {
        'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
      }
    }).then(res => {
      if (res.status === 400 || res.status === 401 || res.status === 403) {
        this.delay = 2*this.delay
        return new Promise(function(resolve, reject) {
          resolve({})
        })
      } else {
        return res.json()
      }
    }).then((snapshot)=>{
        let updatedErrors = snapshot.errors || []
        if (!deepEqual(updatedErrors, collection.errors)) {
          collection.errors = updatedErrors
          collection.notify()
        }
        collection.reconcile(snapshot.projects || [])
      })
      .catch(err => console.log('Error fetching projects: ', err));
  }

  strip(yaml) {
    super.strip(yaml)
    delete yaml.children
  }

}

export class ProjectCollection extends IResourceCollection {

  /* extend */
  constructor() {
    super(new ProjectStore(ProjectResource));
    this.store.subscribe(this);
    this.errors = []
  }

}

/**
 * The AllProjects object manages every Project instance and synchronizes the list of Projects that
 * are instantiated with the real world of Kubernetes.
 */
export var AllProjects = new ProjectCollection();
