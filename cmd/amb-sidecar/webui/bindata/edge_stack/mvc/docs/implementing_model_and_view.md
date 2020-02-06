# Implementing a new Model + View

The best way to understand how to implement a new Model and View is to read the exemplar model and view
classes:

1. We start with the model class for a single Host: [HostResource](../models/host_resource.js). You'll see
that it overrides, and overrides+extends, the "must be implemented by a subclass" methods defined in `IResource`.
In addition, `HostResource` defines additional methods that are special-purpose for the Host model.

2. Next examine the view class for a single Host: [HostView](../views/host_view.js). 
You'll see that it overrides, overrides+extends, the "must be
implemented in a subclass" methods in `IResourceView`.
In addition, `HostView` defines some additional special-purpose
methods to support its DOM elements such as `onACMEcheckboxChanged()`.
You'll also want to pay careful attention to the `renderSelf()` method for how it
supports the dual read-only/writeable DOMs (see [Views and Custom Web-Components](views_and_web_components.md) for
more information).

3. Now that we have the single Host Model (`HostResource`) and View (`HostView`) defined,
we move on to the reading the model for the collection of Hosts: [HostCollection](../models/host_collection.js). 
You'll see that our model class only has to implement two methods
and everything is automatically done by the framework classes.

4. Note that we have a global variable `AllHosts` to hold the
singleton `HostCollection`: since the `HostCollection` maintains
the set of all `HostResource` object, there's no need to have
more than one `HostCollection`.

5. Then we examine the view class for the collection of
Hosts: [HostCollectionView](../views/hostcollection_view.js).
You'll see that our model class only has to implement seven trivial
configuration methods and everything else is done by the framework classes.

6. And finally we define a web-component custom element `dw-mvc-hosts` so
that our `index.html` page can instantiate the `HostCollectionView` at
the correct place in its overall DOM.

That's all there is to implementing a simple view of the collection
of all of a particular kind of CRD. Everything else is handled/implemented
in the internal/hidden framework classes and methods.
