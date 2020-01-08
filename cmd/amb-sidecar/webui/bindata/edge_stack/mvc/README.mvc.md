# Model-View-Controller (MVC)

Model/View/Controller is a design pattern that structures code that:

- separates out the `Model` (the data and data-only behaviors)
- from the `View` (how that model is presented to the user, in potentially many different ways)
- and the `Controller` (how you can change the Model’s state through one or more Views).

MVC was invented at Xerox Palo Alto Research Center in the late 1970’s in the Smalltalk system
and then adopted broadly for other GUI’s and web applications since then (e.g. Spring, Angular, 
React, JQuery Mobile, Vue, etc).  In our usage in the AES Admin UI, due to limitations of HTML 
as well as practicality, the `View` and `Controller` are combined into a single `View` object.

In MVC, a `Model` is an object with state that has Listeners that are interested in being
notified on certain changes in that state.  Any object may listen for changes to a `Model`.
The objects are called `Models` because they are intended to model the real world, in our
case they represent (model) the state of the AES and Kubernetes.

In the Admin UI, `Views` are Listeners to their corresponding `Model` objects. The `View` 
defines the UI application logic that both displays the state of its `Model` and allows 
any changes/updates of that `Model`, as required by the application, with validation and
the ability to cancel pending updates (among other functionality).

Some of the benefits of the MVC approach include:
- the ability to test the `Model` separately from the `View`:
  - We can test performance and reliability of the datapath between the backend and the 
    client without any `View` code being involved.
  - We can build new `Views` independent of a functioning `Model`, by creating a mock 
    `Model` with the state and behavior we want, and then building a `View` that represents it.  
    Once the real `Model` is built it can replace the mock `Model`.
  - We can build new `Models` independent of any `View`, and test them independently.  There 
    are a number of cases where a `Model` with its ability to notify Listeners is useful
    apart from the `Model`/`View` usage.
  
- easier code maintenance and upgrading because it uses the single-responsibility design principle.
  - All state fetch, store, and modification is localized in the `Model` code.
  - `Views` are only responsible for rendering that state
  
- easier development on both `Model` and `View` code:
  - We can add new types of `Model` objects, and new `Views` on existing models, independently.
  - We can have multiple simultaneous `Views` on a single `Model`.
  - A single `View` can incorporate data from multiple `Models`.

## The rationale for using MVC to implement our Admin UX

### Benefits

The MVC implementation of `Resource` and `ResourceView` provide the building blocks for displaying, adding, and editing
Kubernetes Resources, most importantly the CRD's (Custom Resource Definitions) that Ambassador uses to get input
from users and communicate back to users.

There several different goals of using this kind of abstraction:

- Consistency of experience for cross-cutting aspects of CRD's.  One of the things that makes Kubernetes powerful
for advanced users is the fact that that they can treat all the resources the same way.  Names, namespaces,
labels, annotations, and status are some of the examples of shared concepts for which we want to provide
a consistent experience.

- GitOps workflow.  A particularly important example of the uniform handling of `Resource` objects is the
ability to use a GitOps workflow to manage your Kubernetes resources. For example, defining your "source of truth"
declaratively in `git`, and updating your cluster via `kubectl apply` makes it easy to localize changes and
simplify your cluster management.  We need our UI to work well with this GitOps-style workflow.
 
 - Ease of extension to new types of CRD's.  Since all `Resource` objects have a shared format, new kinds may be easily
defined by subclassing the `IResource` interface class and implementing the required methods that specialize
that `Resource`'s state and behavior.
 
 - Ease of customization of CRD display and editing.  With MVC we can easily customize each Resource object
 (e.g. Host, Mapping, etc.) and its corresponding ResourceView so that it can be created, displayed,
and edited in the best way for that particular Resource type.  With built-in links to other relevant resources
we can make naviation much easier and in general, help new users become advanced users much faster than before.
 
### Resource, ResourceView, and ResourceListView
 
There are two Model interfaces (`IResource` and `ICollection`) and two corresponding View interfaces
(`IResourceView` and `IResourceListView`) that are extended via concrete class implementations to define web
components that work with each other.  `IResourceView` is a view on a single `Resource` and `IResourceListView`
is a view on a `Collection` of `Resources`, both web components that work with each other.  For example, a
`HostView` would extend `IResourceView` and define the layout and interaction behavior for a `HostResource`.
A list of hosts, a `HostListView`, is implemented by extending `IResourceListView`, whose model is an instance
of `HostCollection`.
 
### New Functionality
 
There are a number of future features we expect to be adding to the fundamental Model and View specializations
(e.g. `ResourceListViews` and their models:

- Searching/sorting/filtering can be done based on the Kubernetes metadata that is common to all `Resources`
(`name`, `namespace`, `labels`, `annotations`), and custom searching/sorting/filtering for specific kinds.

- Selection of a number of `Resources` and export the yaml.

- Editing of a specific `Resource` and, instead of saving to Kubernetes, downloading the modified YAML.

- Leveraging Kubernetes generate-id to avoid read/modify/write hazards when you edit/save a resource.

- Showing all `Resources` with a non-green status to show prominently on the dashboard.

    - Disallowing editing of Resources that were not created in the UI, so that we never try to write to
Resources that are maintained via GitOps.

- Attaching a URI to a `Resource` that originates from `git`, so that the user can navigate directly to the
`Resource` in the `git` repository from the `Resource` view.

- Leveraging the git repo annotation to allow editing of those `Resources` by filing a PR.

## Implementation Details

The following describes the framework, interfaces, and example classes using the Admin MVC approach.


### The mvc/ directory hierarchy

```
mvc            - toplevel directory, under edge_stack
  framework    - Classes that define the MVC fundamental state and behavior
  interfaces   - Interface classes.  Subclass these for new Resource types and new Views
  models       - The user/developer code, implementing the interfaces
  tests        - unit tests for the new models and views
  views        - View classes based on LitElement and using Models and Collections
```

#### framework

The basic functionality for `Model`, `Collection` and `Resource` classes is defined in
this directory. These are the internal classes of the framework and should not need to
be modified or overridden.
 

#### interfaces

These classes define the interface that must be implemented in concrete subclasses.

Interface classes only define the methods for subclasses to implement. Interface classes do 
not define any behavior in these methods. All interface classes begin with a capital I to 
distinguish from concrete classes that do define state and behavior.

Kubernetes CRD resources will be represented by subclass implementations of `IResource`, 
such as `HostResource` representing a Host CRD.  Similarly, collections of these resources 
will be implemented by subclassing `ICollection`, such as `HostCollection` (a collection 
of `HostResource` objects).

#### models

User/developer code goes here for models, subclasses of `IResource` and `ICollection`.  
For example, the `HostResource` and `HostCollection` classes are models of the Hosts CRDs
and are useful for understanding how one writes concrete implementations of `IResource`
and `ICollection` classes.

#### tests

This directory contains code for testing `Model` and `View` implementations, as well as mock 
implementations of required external functionality and example data, such as the snapshot
service.  (TBD)

#### views

User/developer code goes here for `Views`, subclasses of `LitElement`.  (TBD in future PR's)

### Class definitions

The following are the basic classes that make up the MVC foundation.

As previously mentioned, developers will be subclassing the interfaces `IResource` and `ICollection`, 
implementing the methods that are required.

#### The MVC Class Hierarchy

The following is the class hierarchy, starting with `Model`, and including both concrete and interface classes.

```
Model                  - implements the behavior for notifying listeners
  Resource             - implements basic Resource state and behavior
    IResource          - defines the interface for extending to new Resource kinds
      HostResource     - a concrete implementation of a Host resource
      
  Collection           - implements the behavior and state for maintaining a collection of unique Resources
    ICollection        - defines the interface for extending to new Collections of Resources
      HostCollection   - a concrete implementation of a HostResource Collection

View                   - implements basic behavior: handling Model notifications and rendering. 
  ResourceView         - implements basic behavior, standard Resource state. 
    IResourceView      - defines the interface for extending Resource reading, writing, and validation
      HostView         - a concrete implementation of a HostResource view

CollectionView         - implements basic behavior: handling Model notifications, rendering a sortable list of Views.
  ICollectionView      - defines the interface for extending the CollectionView
```


#### Model and its subclasses

The following simply provides an overview of the actual implementations of `Model`, `Resource`, and `Collection`,
and their interface classes `IResource` and `ICollection`.  Users will typically need only to subclass from
`IResource` and `ICollection`; the framework and interface classes will not be modified.

For more detail on these implementations, see the source code in the `mvc/framework` and `mvc/interfaces`
directories.

##### Model
The `Model` class simply defines methods for managing a group of `Listeners` that may be notified when desired.
As a framework class, this will not be subclassed by the user.

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

##### Resource
The `Resource` class extends the `Model`, so it can have `Listeners`, and it adds state that is common among all
`Resources` (e.g. `kind`, `name`, `namespace`, etc.) and methods for updating its state from snapshot data,
constructing YAML for communication with Kubernetes, and validation of its internal instance variables.

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

##### IResource

The `IResource` interface is subclassed when defining a new `Resource` class.  It is quite simple, 
requiring only methods for updating state, constructing a Kubernetes Spec, and validation.  

```
class IResource extends Resource {
  constructor(data)
  updateSelfFrom(data)
  getSpec()
  validateSelf()
}
```

##### Collection

The `Collection` class extends the `Model`, so it can have `Listeners`.  It subscribes to the snapshot service, 
extracts data from the snapshot when notified, and creates, modifies, or deletes `Resource` objects that it maintains
in the `Collection`.

```
class Collection extends Model {
  constructor()
  onSnapshotChange(snapshot)
}
```

##### ICollection

The `ICollection` interface is subclassed when defining a collection of a specific kind of `Resource`.  It requires
subclasses to identify the class of the Resources in the collection (e.g. a Host), to be able to create a special
string key from snapshot data that is unique for that individual `Resource` instance, and to extract data from
the snapshot to pass to a `Resource` constructor for initializing a new `Resource`, or to an existing `Resource` in
the collection for updating that `Resource`'s state.

```
class ICollection extends Collection {
  constructor()
  resourceClass()
  uniqueKeyFor(data)
  extractDataFrom(snapshot)
}
```

#### View and its subclasses

The following simply provides an overview of the actual implementations of `View`, `ResourceView`, and
`ResourceListView`,  and their interface classes `IView` and `IResourceView`.  Users will typically need
only to subclass from `IResourceView` and `IResourceListView`; the framework and interface classes will not
be modified.

For more detail on these implementations, see the source code in the `mvc/framework` and `mvc/interfaces`
directories.

##### View
The `View` class...
As a framework class, this will not be subclassed by the user.

##### IView
The `IView` class...

##### ResourceView
The `ResourceView` class...
As a framework class, this will not be subclassed by the user.

##### IResourceView
The `IResourceView` class...

# Examples

See `HostResource` and `HostCollection` in mvc/models for specific details on implementation.

# TO DO

HostView, HostListView examples (TBD)
