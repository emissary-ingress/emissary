# MVC: Models

describe a bit what a `Model` is and how it fits into the scheme of things

## What is a Model?

an object that has listeners
listeners define a single method, `onModelNotification`
views listen to individual resources (e.g. `HostResource`)
views also listen to collections of resources (e.g. `HostCollection`)

## The mvc/ directory hierarchy

```
mvc            - toplevel directory, under edge_stack
  framework    - Classes that define the MVC fundamental state and behavior
  interfaces   - Interface classes.  Subclass these for new Resource types and Collections
  models       - The user/developer code, implementing the interfaces
  tests        - unit tests for the new models and views
  views        - View classes
```

#### Framework

basic functionality for model, collection and resource classes is defined
here.

#### Interfaces

These classes define the interface -- the required methods -- that must be implemented in concrete subclasses.
Kubernetes CRD resources will be represented by subclass implementations of `IResource`, such as `HostResource`
representing a Host CRD.  Similarly, collections of these resources will be implemented by subclassing
`ICollection`, such as `HostCollection` (a collection of `HostResource` objects).

# Class Definitions

The following are the basic classes that make up the MVC foundation.

Developers will primarily be subclassing `IResource` and `ICollection`, implementing the interfaces that are required,
but may also call methods from their subclasses that are defined in the framework classes
`Model`, `Resource`, and `Collection`.

### The MVC Class Hierarchy

```
Model                  - defines the behavior for notifying listeners
  Resource             - defines basic Resource state and behavior
    IResource          - defines the interface for extending to new Resource kinds
      HostResource     - a concrete implementation of a Host resource
      
  Collection           - defines the behavior and state for maintaining a collection of unique Resources
    ICollection        - defines the interface for extending to new Collections of Resources
      HostCollection   - a concrete implementation of a HostResource Collection
```


### Model subclasses

```
class Model {
  constructor()
  addListener(listener, messageSet)
  removeListener(listener, messageSet)
  notifyListeners(notifyingModel, message, parameter)
  notifyListenersUpdated(notifyingModel)
  notifyListenersCreated(notifyingModel)
  notifyListenersDeleted(notifyingModel)
}
```

```
class Resource extends Model {
  constructor(data)
  updateFrom(data)
  getEmptyStatus()
  getYAML()
  sourceURI()
  validate()
  validateName(name)
  validateEmail(email)
  validateURL(url)
}
```

```
class IResource extends Resource {
  constructor(data)
  updateSelfFrom(data)
  getSpec()
  validateSelf()
}
```


```
class Collection extends Model {
  constructor()
  onSnapshotChange(snapshot)
}
```

```
class ICollection extends Collection {
  constructor()
  resourceClass()
  uniqueKeyFor(data)
  extractDataFrom(snapshot)
}
```


# Examples

See `HostResource` and `HostCollection` in mvc/models.
