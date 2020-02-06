# Interfaces and Classes

Our MVC framework is divided into three parts:
1. **Interfaces** are classes that define the methods that you must implement in order to participate in the
the framework.
2. **Framework Classes** are the internal implementation classes, hidden by the scenes, that you don't need
to know about but, being the curious developer you are, you surely do want to know about them so this documentation
explains how they work (even though you don't need to know).
3. **Your Model and View Classes** are your concrete implementations of the various interfaces for your
specific CRDs and UI views and florishes.

## Interfaces

One of the keys to understanding our MVC framework is understand interfaces and the relationship between interface,
the framework classes, and your classes. An interface is a class that defines the set of methods that your
concrete class must implement. For example, the `IDonut` interface says that in order to be an `IDonut`, your
class must implement `flavor()` and `size()`:

    class IDonut {
       flavor() { ... }
       size() { ... }
    }

Note that to make sure that we catch incomplete implementations, the methods in the interface often throw errors
like this:

    class IDonut {
       flavor() { throw new Error("must implement"); }
       size() { throw new Error("must implement"); }
    }

Thus if you make a subclass but forget to implement one of the methods, the code will throw an error. For example,
this is a good donut class:

    class AppleFritter extends IDonut {
       flavor() { return "apple"; }
       size() { return "larger"; }
    }

But this is an incorrect subclass (because it is an incomplete implementation):

    class JellyDonut extends IDonut {
       flavor() { return "jelly"; }
    }

All interface classes begin with a capital I to distinguish them
from concrete classes that define state and behavior.

### Override versus Extend

Some of the methods in the interface are designed to be 100% overridden whereas others are designed to
be extended. You, as the concrete subclass developer, should implement both types of methods in your
class, but for the latter you must call `super.foo()` in your method.

You can tell the difference between these two types because the designed-to-be-overridden methods throw
an error (to force you to override them) whereas the designed-to-be-extended methods call their superclass's
method. For example:

    class IDonut {
       flavor() { throw new Error("must implement"); }
       isStale() { return super.isStale(); }
    }

Thus your concrete donut class would do something like this:

    class ChocolateCakeDonut {
       flavor() { return "chocolate"; } /* override */
       isStale() { if( Monday ) return true; else return super.isStale(); } /* extend */
    }


## Framework Classes

Most of the hard-work in the MVC framework happens in internal framework classes (obviously). Most of these
classes are related to the interfaces but we've deliberate separated the interface that you must implement from
the internal methods that the framework implements. Some frameworks expose both kinds of methods (interface and
internal) in a single class, but we find that confusing -- Do I need to override this method or not...? -- so we
keep the two classes separate. This leads to a hierarchy like this:

    InternalClass   # internal implementations that you do not override
      Interface     # only the methods that you must implement
         YourClass  # your implementation of those required methods

The net result of this is that `YourClass` has your methods and all the internal implementation methods and thus
can do all the MVC things that it needs to do.

### Extended Interfaces

To help you understand which methods from `InternalClass` you might want to use in `YourClass`, we often 
document those methods in the interface class. These additional methods are not required-to-be-implemented
methods, obviously, but are in the interface class for your convenience. For example:

    class IDonut {
       /* === must be implemented === */
       flavor() { throw new Error("must implement"); }
       size() { throw new Error("must implement"); }
       /* === useful methods you might want to use === */
       weight() { return super.weight(); }
       calories() { return super.calories(); }
    }

Having these methods in the interface class means that you don't have to look into the internal
classes at all. (But, knowing that you're a curious developer, I'm sure you will anyway.)

## Corresponding Models and Views

Model objects are the models of the Ambassador and Kubernetes state. Currently all of our model objects are
models of CRDs.
View objects are the views of those models.
Currently, there is a one-to-one correspondence between models and views.

There are two Model interfaces (`IResource` and `IResourceCollection`) and two corresponding View interfaces
(`IResourceView` and `IResourceCollectionView`) that you extend to create corresponding models and views for
the CRDs:  

|  MODEL  | VIEW  | PURPOSE | 
|:---------:|:--------:|-----|
| IResource | IResourceView | model and view for a single CRD |
| IResourceCollection  |  IResourceCollectionView | model and view for a collection of CRDs

For example, the models and views for a single Host CRD and a collection of Host CRDs would be:

|    | MODEL | VIEW  |
|----|:-----:|:-----:|
| single Host | `HostResource(IResource)` | `HostView(IResourceView)` |
| collection of Hosts | `HostCollectionView(IResourceCollectionView)` | `HostCollection(IResourceCollection)` |

### Class Hierarchy

When you look at the whole inheritance hierarchy, including the internal framework classes, you end up
with this:

    +--------------+    # the internal framework
    |     Model    |    #  class that provides
    +--------------+    #  behavior for notifying listeners
            ^
            |
    +--------------+    # the internal framework
    |   Resource   |    #  class that provides basic
    +--------------+    #  Resource state and behavior
            ^
            |
    +--------------+    # the interface class
    |  IResource   |    #  that defines what methods
    +--------------+    #  your subclass must implement
            ^
            |
    +--------------+
    | HostResource |    # your subclass
    +--------------+
    

Here's a more complete listing for all four of the interfaces and your model and view classes:

```
Model                     - implements the behavior for notifying listeners
  Resource                - implements basic Resource state and behavior
    IResource             - defines the interface for extending to new Resource kinds
      HostResource        - a concrete implementation of a Host resource
      
  ResourceCollection      - implements the behavior and state for maintaining a collection of unique Resources
    IResourceCollection   - defines the interface for extending to new ResourceCollections
      HostCollection      - a concrete implementation of a HostResource ResourceCollection

View                      - implements basic behavior: handling Model notifications and rendering. 
  ResourceView            - implements basic behavior, standard Resource state. 
    IResourceView         - defines the interface for extending Resource reading, writing, and validation
      HostView            - a concrete implementation of a HostResource view

ResourceCollectionView    - implements basic behavior: handling Model notifications, rendering an optionally sortable list of Views.
  IResourceCollectionView - defines the interface for extending the ResourceCollectionView
    HostCollectionView    - implements the Host-specific ResourceCollectionView behavior
```
