MOREMORE still to be moved to the correct file





# How Resources are Added, Deleted, and Edited

The process of deleting a `Resource` from a `ResourceCollectionView`, adding a new `Resource`, and editing an existing
`Resource` must take into account the ongoing update process from snapshots arriving with new data.  These different
operations require functionality to change the `ResourceView`'s appearance using its `viewState` property and to notify
the `ResourceCollection` of pending operations on the `Resource` being added, deleted, or modified using the Resource's
`pending` property.

### The ResourceView's viewState

`ResourceView` is responsible for rendering a single `IResource`.  Subclasses of `IResourceView`,
such as `HostView`, will implement specific behavior and rendering for that type of `IResource`.

The process of viewing, editing, adding and deleting `IResource` in the `ResourceCollectionView` requires a number of
different renderings.  These renderings are controlled by the `viewState` property on the `ResourceView`.


| `viewState` | purpose | display |
|--------|--------------------------------|-------|
| `list` | when viewing the `IResource` in the `IResourceCollectionView` | divs and spans |
| `edit` | when editing an existing `IResource`'s attributes | form fields |
| `add` | when creating a new `IResource` that doesn't already exist in the `IResourceCollection` | form fields |
| `pending` | after the user has clicked on "save" after an add or edit operation, but before Kubernetes has completed the operation | like `list` but with a cross-hatched background pattern |

Because viewState is a `LitElement` property, setting the viewState to a value will cause the `ResourceView` to be
re-rendered.

### The Resource pending flag

The `pending` state requires a little explanation: the user makes changes to the UI immediately and expects instant
feedback that he or she had made a change, but the Kubernetes back-end is updated asynchronously and may even fail
to accept a particular change. Thus we need a way for the UI to indicate to the user that his or her change is
in the process of being made, a state that we call `pending`.

MOREMORE

The Resource class's `_pending` flag is a way to tell the `ResourceCollection` that a given `Resource` in the collection
needs to be handled differently.  The `_pending` flag can be set to three different string values:

- `add`, when the `Resource` has been added but not yet confirmed by existence in a snapshot;
- `save`, when the `Resource` has been edited but its new state has not yet been seen in a snapshot;
- `delete`, when the `Resource` has been deleted from the system and is awaiting a snapshot confirming the deletion.

The `ResourceCollection` checks the `_pending` flag on the `Resource` when it processes a new snapshot.  If the `Resource`
is in the `ResourceCollection` already, and the existing `Resource`'s `resourceVersion` is not the same as the
`resourceVersion` seen in the snapshot, then the `Resource` is updated from the snapshot data and any `_pending`
flag is cleared.

If the `Resource` in the collection has been added by the user, then it will be in the `ResourceCollection` but not
necessarily observed in a snapshot.  Normally in this case the `Resource` would be deleted and its listeners notified.
But in the case of being added, the `ResourceCollection` checks the pending flag and does not delete the object if
the addition is pending.  At some point the snapshot will show the `Resource` as existing in the system, the `Resource`
will be updated with its status, and the `_pending` flag will again be cleared.

However, if the backend takes too long or fails to add, save, or delete the `Resource` for some reason, the
timer will time out and clear the flags, cancelling the operation.  With the flags cleared:
 
 - if pending `add`, the `Resource` will be removed from the `ResourceCollection` at the next snapshot cycle;
 - if pending `save`, the `Resource` will be restored to its original state;
 - if pending `delete`, the `Resource` will still be represented in the snapshot and not removed.

This will return the system to a consistent state.

## Deleting an existing Resource

Deleting an existing `Resource` (e.g. a `HostResource`) is the simplest case of the three.  The `Resource` is
rendered by the `ResourceView`, which provides a `Delete` button.  When the button is pressed, the following will occur:
- the model (e.g. the `HostResource`) will be sent a `doDelete` message, which composes a request to the edge stack with
its resource `kind`, `name`, and `namespace`, and a `delete` action.
- if the request to the edge stack to delete the resource fails or returns an error, then an alert is shown
displaying the error to the user, and the `ResourceView` changes its `viewState` back to `list`.
- otherwise, the `Resource` sets its `_pending` state to `delete`;
- the `ResourceView` sets its `viewState` to `pending`, which shows a pattern over the view, and
a `Pending` button is displayed;
- A timer is set for 5 seconds, to check if the operation has succeeded. If the `delete` operation has succeeded,
the timer is ignored.  However, if the `delete` has not succeeded, the timeout is used for restoring the system to
its consistent state.

At this point the system is awaiting one of two outcomes: the `Resource` has been successfully removed from the
system, or a timeout.

The succcessful outcome: at some future time the snapshot will show that the `Resource` no longer exists.  Then:
- the `ResourceView` is notified and will then remove itself from the `ResourceCollectionView`.

The timout fires: this means that the `ResourceView` has not been notified within 5 seconds that the `Resource`
no longer exists.  Then:
- the Resource clears its `_pending` flag.
- the `ResourceView` resets its `viewState` to `list`.
- an alert is shown, notifying the user that the delete failed.

## Adding a new Resource

Adding a new `Resource` (e.g. a new `HostResource`) needs to check whether the new `Resource` being added conflicts in
some way with an existing `Resource` in the system.  In Kubernetes, the triple `(kind, name, namespace)` is unique
within the system, so any new `Resource` being added must not already have the same values for those attributes.

The `ResourceCollectionView` is responsible for adding new `Resources`.  When the `Add` button is pressed,
the following will occur:
- the `ResourceCollectionView` will create a new instance of the appropriate `Resource` type (e.g. a new
`HostResource`) and its pending state set to `add`.
- the `ResourceCollectionView` will create a new `ResourceView` of the appropriate type (e.g. `HostView`)
and initialized with the `Resource` instantiated above as its `model`.
- the `ResourceCollectionView` will insert the new `ResourceView` at the beginning of its list of `ResourceViews`;
- the `onAdd()` method for the `ResourceView` will be called.
- The `ResourceView` will set its `viewState` to `add` and request the focus.  This will change the view to an
edit mode, where the user may then change the `Resource's` attribute values.

When the user is done and clicks `Save`:
- the `ResourceView` and `Resource` will validate their state for correctness and uniqueness.  Because this `Resource`
is being added, it must not have the same `(kind, name, namespace)` as any existing `Resource` in the system.
If there are any errors, they will be added to the message section of the `ResourceView`.
- If the fields are validated, then the `doSave()` method is invoked on the new `Resource`, which sends a request
to the edge stack to create its new `Resource` object in the backend.
- If the request to the edge stack to add the resource fails or returns an error, then an alert is shown
displaying the error to the user, and the `ResourceView` changes its `viewState` back to `list`.
- otherwise, the `Resource` is added to the `ResourceCollection`, pending confirmation of its being added to the
system by being represented in a future snapshot.  The `Resource`'s pending `add` state tells the `ResourceCollection`
not to delete the `Resource` even if it isn't represented in a snapshot.
- A timer is set for 5 seconds, to check if the operation has succeeded. If the add operation has succeeded,
the timer is ignored.  However, if the add has not succeeded, the timeout is used for restoring the system to
its consistent state.

At this point the system is awaiting one of two possible outcomes.

The succcessful outcome: at some future time the snapshot will show that the `Resource` has been added. Then:
- the `ResourceCollection` will update the `Resource` and notify the `ResourceView`;
- the `Resource` will clear its `_pending` flag;
- the `ResourceView` will reset its `viewState` to `list`.

The timeout fires.  Then:
- the `Resource` clears its _pending flag, which will allow the `ResourceCollection` to remove the `Resource` at the
next snapshot update;
- and the `ResourceView` changes its `viewState` back to `list`.




-----------------------------------------


## Implementation Details

The following describes the framework, interfaces, and example classes using the Admin MVC approach.

### The mvc/ directory hierarchy

```
mvc            - toplevel directory, under edge_stack
  interfaces   - Interface classes.  You will subclass these for new types of Resources and Views
  models       - Your models, based on IResource and IResourceCollection
  views        - Your views, based on IResourceView and IResourceCollectionView
  tests        - Unit tests for the new models and views
  framework    - Classes that define the MVC fundamental state and behavior
```

#### models

Your code for models goes here; your models are subclasses of `IResource` and `IResourceCollection`.  
For example, the `HostResource` and `HostCollection` classes are models of the Hosts CRDs
and are useful for understanding how one writes concrete implementations of `IResource`
and `IResourceCollection` classes.

#### views

Your code for views goes here; your views are subclasses of `IResourceView` and `IResourceCollectionView`.
For example, the `HostView` and `HostCollectionView` are views on `HostResource` and `HostCollection`
respectively, and are useful for understanding how one writes concrete implementations of views to show
`IResource` and `IResourceCollection`.

#### tests

This directory contains code for testing `Model` and `View` implementations, as well as mock 
implementations of required external functionality and example data, such as the snapshot
service.  (TBD)

#### framework

The basic framework classes `Model`, `Resource`, `View`, `ResourceView`, `ResourceCollection`,
and `ResourceCollectionView` are defined in this directory. These are the internal classes of the framework
and should not need to be modified or overridden.
 
### Class definitions

The following are the basic classes that make up the MVC foundation.

As previously mentioned, developers will be subclassing the interfaces `IResource` and `IResourceCollection`, 
implementing the methods that are required.

#### How We Use Interfaces

In order to keep a clean separation between the interface specification (the methods your subclass must implement)
and the internal framework implementations (the methods the framework implements behind the scenes), we have divided
each conceptual class into two classes: the interface class and the framework class.
  
When you write a class, for example `HostResource`, that implements an interface, you do that by extending/subclassing
the interface `IResource`:

    class HostResource extends IResource { ... }


#### Model and its subclasses

The following simply provides an overview of the actual implementations of `Model`, `Resource`, and `ResourceCollection`,
and their interface classes `IResource` and `IResourceCollection`.  Users will typically need only to subclass from
`IResource` and `IResourceCollection`; the framework and interface classes will not be modified.

For more detail on these implementations, see the source code in the `mvc/framework` and `mvc/interfaces`
directories.

##### Model
The `Model` class simply defines methods for managing a group of objects (listeners) that may be notified when 
something changes in the model. Listeners must implement the `onModelNotification(model, message, parameter)`
method. The meaning of the notification call is:
 * `model` has `message`'d and you might want this optional `parameter`
 * for example, the model "localhost" has been "created"

As a framework class, this class will not be subclassed directly.

```
class Model {
  addListener(listener, messageSet)
  removeListener(listener, messageSet)
  notifyListeners(notifyingModel, message, parameter)
  notifyListenersUpdated(notifyingModel) // notify listeners of "updated"
  notifyListenersCreated(notifyingModel) // notify listeners of "created"
  notifyListenersDeleted(notifyingModel) // notify listeners of "deleted"
}
```

##### Resource
The `Resource` class is a `Model`, so it can notify its listeners of any suitable events, and it adds state
that is common among all `Resources` (e.g. `kind`, `name`, `namespace`, etc.) as well as methods for updating its
state from snapshot data, constructing YAML for communication with the edge stack, and validation of its internal
instance variables.

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
  getYAML()
  validateSelf()
}
```

##### ResourceCollection

The `ResourceCollection` class is `Model`, so it can notify its listeners.  It subscribes to the snapshot service, 
extracts data from the snapshot when new snapshots arrive, and it creates, modifies, or deletes the `Resource` objects 
that it maintains in its collection.

The `ResourceCollection` contains a collection of unique `IResource` objects. The uniqueness is determined by the 
`IResourceCollection.uniqueKeyFor()` method. When the `ResourceCollection` scans the snapshot data, if it finds
a unique key that it does not yet have, it creates an `IResource` object for that key; if it finds a unique key
that it already has, it updates that `IResource` object (if needed); and if it does not find a unique key in the
snapshot for an `IResource` that it currently holds, then it deletes that object from the collection. In each case
(of create, update, and delete), it notifies its listeners of that change (its listeners are usually views which
will then update themselves and thus the display).

```
class ResourceCollection extends Model {
  onSnapshotChange(snapshot)
  addResource(resource)
  hasResource(resource)
}
```

##### IResourceCollection

The `IResourceCollection` interface is subclassed when defining a new `ResourceCollection` for a specific `Resource`.
It requires subclasses to identify the class of the Resources in the collection (e.g. a Host), to be able to create
a special string key from snapshot data that is unique for that individual `Resource` instance, and to extract data from
the snapshot to pass to a `Resource` constructor for initializing a new `Resource`, or to an existing `Resource` in
the collection for updating that `Resource`'s state.

```
class IResourceCollection extends ResourceCollection {
  extractResourcesFrom(snapshot)
  resourceClass()
  uniqueKeyFor(resource)
  ...
}
```

#### View and its subclasses

The following simply provides an overview of the actual implementations of `View`, `ResourceView`, and
`ResourceCollectionView`,  and their interface classes `IResourceView` and `IResourceCollectionView`.
Users will typically need only to subclass from `IResourceView` and `IResourceCollectionView`; the framework
and interface classes will not be modified.

For more detail on these implementations, see the source code in the `mvc/framework` and `mvc/interfaces`
directories.

##### View
The `View` class defines the basic HTML framework, rendering, and model notification handling for display of
a single `Resource` (e.g. a `HostResource`).  The `render()` method here assumes styles are imported properly
and a common layout of the view that has `list`, `edit`, and `add` variants.  The developer
should not have to subclass this but would instead subclass `IResourceView` and implement the methods
required there.

```
class View extends LitElement {
  static get properties()
  constructor(model)
  modifiedStyles()
  onModelNotification(model, message, parameter)
  render()
  visibleWhen(...arguments)
}
```

##### ResourceView
The `ResourceView` class is a `View` subclass that adds `Resource`-specific state and handling for rendering a single
`Resource` object.  It handles the edit operations (`edit`, `save`, `cancel`) and rendering of the different variants of
the `ResourceView` in these view states.  It also provides the ability to add messages to the end of the `View` as well
as an optional YAML display showing the YAML that represents the resource.  When editing, the YAML will display the
actual structure that the edge stack would send to Kubernetes apply.

The developer will not subclass `ResourceView` but instead `IResourceView`, implementing the required methods there.

```
class ResourceView extends View {
  static get properties()
  constructor(model)
  addMessage(message)
  clearMessages()
  nameInput()
  namespaceInput()
  onCancelButton()
  onEditButton()
  onSaveButton()
  onSource(mouseEvent)
  readFromModel()
  writeToModel()
  render()
  renderMessages()
  renderYAML()
  validate()
}
```

##### IResourceView
The `IResourceView` class is the interface class that developers will subclass for their specialized views
for `Resources`.  Because most of the functionality of rendering, editing and updating `Resources` is handled in the
concrete classes `ResourceView` and View, the developer need only implement the methods in the interface.  For
an example, see `HostResourceView`, which extends `IResourceView`.

```
class IResourceView extends ResourceView {
  static get properties()
  constructor(model)
  readSelfFromModel()
  writeSelfToModel()
  renderSelf()
  validateSelf()
```

##### ResourceCollectionView
The `ResourceCollectionView` class implements the display of a list of `IResourceView` subclasses
(e.g. `HostResourceView`).  It listens for notifications from a `ResourceCollection` subclass (e.g.
`HostCollection`) which manages the list of `IResources` (e.g. `HostResources`) and creates and deletes the
appropriate `IResourceViews` as needed.  Irt also provides sorting of these views by an attribute of the
`IResources` being displayed (e.g. the `HostResource`'s `name`, `namespace`, or other attribute).

Developers will not subclass `ResourceCollectionView` but instead will subclass `IResourceCollectionView` and 
implement the required methods there.

```
class ResourceCollectionView extends LitElement {
  static get properties()
  static get styles()
  constructor()
  onAddButton()
  onChangeSortByAttribute(e)
  onModelNotification(model, message, parameter)
  readOnly()
  render()
}
```

##### IResourceCollectionView
The `IResourceCollectionView` class is the interface class that developers will subclass for their specialized
list views of `Resources`.  The developer need only define the methods listed below.  For a concrete example,
see `HostCollectionView` which implements a concrete class displaying a list of `HostResource` objects.

```
class IResourceCollectionView extends ResourceCollectionView {
  static get properties()
  static get styles()
  constructor()
  readOnly()
  viewClass()
}

```
