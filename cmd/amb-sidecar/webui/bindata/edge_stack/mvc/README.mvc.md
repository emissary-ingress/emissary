# Model-View-Controller (MVC)

Model/View/Controller is a design pattern that structures code that:

- separates out the `Model` (the data and data-only behaviors)
- from the `View` (how that model is presented to the user, in potentially many different ways)
- and the `Controller` (how you can change the Model’s state through one or more Views).

MVC was invented at Xerox Palo Alto Research Center in the late 1970’s but has been adopted for GUI’s and web
applications, and the pattern used broadly since then (Spring, Angular, React, JQuery Mobile, Vue).  In our
usage in the Admin UI, due to limitations of HTML as well as practicality, the `View` and `Controller` are not
separate objects; `Controller` functionality is simply combined in with View functionality.

A `Model` is an object with state that has `Listeners` that are interested in being notified on certain changes
in that state. A model can add and remove listeners, and listeners can specify which messages they are interested
in (or alternatively, all messages can be sent). 

Any object may be a `Listener`.  An object that is a `Listener` need only define a single method, `onModelNotification`,
which is called when the model is sending a message to which that `Listener` has subscribed.

In the Admin UI, `Views` are `Listeners` to their corresponding `Model` objects. The `View` defines the UI
application logic that both displays the state of its `Model` and allows any changes/updates of that `Model`,
as required by the application, with validation and the ability to cancel pending updates (among other
functionality).

Some of the benefits of the MVC approach include:
- the ability to test the `Model` separately from the `View`:
  - We can test performance and reliability of the datapath between the backend and the client without any View
  code being involved.
  - We can build new `Views` independent of a functioning `Model`, by creating a mock `Model` with the state
  and behavior we want, and then building a `View` that represents it.  Once the real `Model` is built it
  can replace the mock `Model`.
  - We can build new `Models` independent of any `View`, and test them independently.  There are a number of cases
  where a `Model` with its ability to notify `Listeners` is useful apart from the `Model`/`View` usage.
  
- easier code maintenance and upgrading because it uses the single-responsibility design principle.
  - All state fetch, store, and modification is localized in the `Model` code.
  - `Views` are only responsible for rendering that state (or selected parts of the state)
  
- easier development on both `Model` and `View` code:
  - We can add new types of `Model` objects, and new `Views` on existing models, independently.
  - We can have multiple simultaneous `Views` on a single `Model`.

## The mvc/ directory hierarchy

```
mvc            - toplevel directory, under edge_stack
  framework    - Classes that define the MVC fundamental state and behavior
  interfaces   - Interface classes.  Subclass these for new Resource types and Collections
  models       - The user/developer code, implementing the interfaces
  tests        - unit tests for the new models and views
  views        - View classes based on LitElement and using Models and Collections
```

#### framework

basic functionality for model, collection and resource classes is defined
here.

#### interfaces

These classes define the interface -- the required methods -- that must be implemented in concrete subclasses.
Kubernetes CRD resources will be represented by subclass implementations of `IResource`, such as `HostResource`
representing a Host CRD.  Similarly, collections of these resources will be implemented by subclassing
`ICollection`, such as `HostCollection` (a collection of `HostResource` objects).

#### models

user/developer code goes here for `Models` and subclasses of `IResource` and `ICollection` (more TBD)

#### tests

unit tests

#### views

user/developer code goes here for `Views`, subclasses of LitElement...

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
