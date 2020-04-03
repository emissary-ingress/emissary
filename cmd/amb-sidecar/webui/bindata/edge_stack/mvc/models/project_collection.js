/**
 * ProjectCollection
 * This is an IResourceCollection subclass for Project objects.
 */

import { ProjectResource } from "./project_resource.js"
import { IResourceCollection } from "../interfaces/iresource_collection.js";
import { ResourceStore } from "../framework/resource_store.js";
import { getCookie } from '../../components/cookies.js';
import { ApiFetch } from "../../components/api-fetch.js";

class ProjectStore extends ResourceStore {

  subscribe(collection) {
    let looper = ()=>{
      this._poll(collection);
      setTimeout(looper, 1000);
    }
    looper();
  }

  _poll(collection) {
    ApiFetch(`/edge_stack/api/projects`, {
      headers: {
        'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
      }
    }).then(res => res.json())
      .then((projects)=>collection.reconcile(projects))
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
  }

}

/**
 * The AllProjects object manages every Project instance and synchronizes the list of Projects that
 * are instantiated with the real world of Kubernetes.
 */
export var AllProjects = new ProjectCollection();
