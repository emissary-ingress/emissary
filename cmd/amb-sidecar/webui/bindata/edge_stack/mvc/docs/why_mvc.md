# Model-View-Controller (MVC)

Model/View/Controller is a design pattern that structures code that:

- separates out the `Model` (the data and data-only behaviors)
- from the `View` (how that model is presented to the user, in potentially many different ways)
- and the `Controller` (how you can change the Model’s state through one or more Views).

MVC was invented at Xerox Palo Alto Research Center in the late 1970’s in the Smalltalk system
and then adopted broadly for other GUI’s and web applications since then (e.g. Spring, Angular, 
React, JQuery Mobile, Vue, etc).  In our usage in the AES Edge Policy Console UI, due to limitations of HTML 
as well as practicality, the `View` and `Controller` are combined into a single `View` object.

In MVC, a `Model` is an object with state that has Listeners that are interested in being
notified on certain changes in that state.  Any object may listen for changes to a `Model`.
The objects are called `Models` because they are intended to model the real world, in our
case they represent (model) the state of the AES and Kubernetes.

In the Edge Policy Console UI, `Views` listen to notifications from their corresponding `Model` objects. The `View` 
defines the UI application logic that both displays the state of its `Model` and provides 
changes and updates of that `Model`, with validation and
the ability to cancel pending updates, among other functionality.

Some of the benefits of the MVC approach include:
- the ability to test the `Model` separately from the `View`:
  - We can test performance and reliability of the datapath between the backend and the 
    client without any `View` code being involved.
  - We can build new `Views` independent of a functioning `Model`, by creating a mock 
    `Model` with the state and behavior we want, and then building a representative `View`.  Once the 
    real `Model` is built, it can replace the mock `Model`.
  - We can build new `Models` independent of any `View`, and test them independently.
  
- easier code maintenance and upgrading because it uses the single-responsibility design principle.
  - All state fetch, store, and modification is localized in the `Model` code.
  - `Views` are only responsible for rendering that state.
  
- easier development on both `Model` and `View` code:
  - We can add new types of `Model` objects, and new `Views` on existing models, independently.
  - We can have multiple simultaneous `Views` on a single `Model`.
  - A single `View` can incorporate data from multiple `Models`.

### Benefits

The MVC implementation of `Resource` (a `Model`) and `ResourceView` (a `View`) provide the building blocks for
displaying, adding, and editing Kubernetes `Resources`, most importantly the CRD's (`Custom Resource Definitions`) that
the Ambassador Edge Stack uses to get input from users and communicate back to users.

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
 
 - Ease of customization of CRD display and editing.  With MVC we can easily customize each `Resource` object
 (e.g. `Host`, `Mapping`, etc.) and its corresponding `ResourceView` so that it can be created, displayed,
and edited in the best way for that particular `Resource` type.  With built-in links to other relevant resources
we can make navigation much easier and in general, help new users become advanced users much faster than before.

 
### Future Functionality
 
There are a number of future features we expect to be adding to the fundamental `Model` and `View` specializations
(e.g. `IResourceCollectionView`) and their models:

- Searching/sorting/filtering can be done based on the Kubernetes metadata that is common to all `Resources`
(`name`, `namespace`, `labels`, `annotations`), and custom searching/sorting/filtering for specific kinds.

- Selection of a number of `Resources` to export the YAML.

- Editing of a specific `Resource` and, instead of saving to Kubernetes, downloading the modified YAML.

- Leveraging Kubernetes generate-id to avoid read/modify/write hazards when you edit/save a `Resource`.

- Showing all `Resources` with a non-green status to show prominently on the dashboard.

- Disallowing editing of `Resources` that were not created in the UI, so that we never try to write to
`Resources` that are maintained via GitOps.

- Attaching a URI to a `Resource` that originates from `git`, so that the user can navigate directly to the
`Resource` in the `git` repository from the `Resource` view.

- Leveraging the git repo annotation to allow editing of those `Resources` by filing a PR.

### Other Bug Fixes and Improvements

- Currently we cannot have multiple views of the same model that automatically stay synchronized because when a
view writes back to its model, the model does not notify any other views of the changes. This will be easy to fix
and is noted in the code with a TODO comment.

- Proper email address and URL validation needs to be implemented.  Currently there is a simple regex for email
and no URL validation at all.  This is noted in the code with TODO.

- There is a potential race condition in `ResourceCollectionView` where the models in the collection are all created
before the `ResourceCollectionView` itself, and thus is not notified to create the corresponding subviews.  This
is fixed by calling the `ResourceCollection` from the `firstUpdated` callback and asking to be notified of all
the models in the `ResourceCollection`, so that the views can be created.  This assumes that the `ResourceCollectionView`
has not been notified of any additions before `firstUpdated` is called; although this has not been observed it is
theoretically possible.  To fix this, the `ResourceCollectionView` should check in `onModelNotification` that the
model being added doesn't yet have a corresponding view.  If it does, the notification should be ignored.  This
is noted in the code with TODO.


