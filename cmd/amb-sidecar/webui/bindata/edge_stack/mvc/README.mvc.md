# MVC: Models

describe a bit what a Model is and how it fits into the scheme of things

## What is a Model?

an object that has listeners
listeners define a single method, onModelNotification
views listen to individual resources (e.g. HostResource)
views also listen to collections of resources (e.g. HostCollection)

## Framework

basic functionality for model, collection and resource classes is defined
here.

## Interfaces

In Ambassador Pro, a _descriptor_ defines what is rate
limited. Descriptors can contain arbitrary metadata about a request.
Ambassador Pro uses this approach instead of using fixed fields (e.g.,
URLs, client IPs, etc.) to give the end user more control over what
exactly is rate limited.

A descriptor is a key/value pair, e.g., `database:users` or
`catalog:*`. Each descriptor is configured to have its own rate limit.


# Class Definitions

The following are the basic classes that make up the MVC foundation.

Developers will primarily be subclassing `IResource` and `ICollection`, implementing the interfaces that are required,
but may also call methods from their subclasses that are defined in the framework classes
`Model`, `Resource`, and `Collection`.

## The MVC Class Hierarchy

```
Model                  - defines the behavior for notifying listeners
  Resource             - defines basic Resource state and behavior
    IResource          - defines the interface for extending to new Resource kinds
      HostResource     - a concrete implementation of a Host resource
      
  Collection           - defines the behavior and state for maintaining a collection of unique Resources
    ICollection        - defines the interface for extending to new Collections of Resources
      HostCollection   - a concrete implementation of a HostResource Collection
```


# Model subclasses

Resource

IResource

Collection

ICollection

# Examples

## HostResource


## HostCollection


