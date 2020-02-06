# UI updates View --> Model --> Ambassador

(1) When the user uses the UI to update the view (via `onSaveButton()`) view B, the simple explanation
is that (2) view B updates model B (via `writeToModel()`), and then (3) model B sends the updates to Ambassador
via an API call with updated YAML (via `doSave()`).

                       [models]         [views]
                      +---------+     +---------+
    [Ambassador]      | +-----+ |-----| +-----+ |
      Host A          | |  A  |---------|  A  | |
      Host B <--+     | +-----+ |     | +-----+ |
      Host Q    |     | +-----+ |     | +-----+ |
                +-(3)---|  B  |<--(2)---|  B  |<-----(1)
                      | +-----+ |     | +-----+ |
                      | +-----+ |     | +-----+ |
                      | |  Q  |---------|  Q  | |
                      | +-----+ |     | +-----+ |
                      +---------+     +---------+

The reality is more complicated because the API call to Ambassador is asynchronous with the UI and relatively
slow from the point of view of the user. Thus we need to show the user a "pending" state while that API is
in process, and we need to handle both happy and sad path results from the asynchronous API call.

Additionally, we don't want the model that the user is editing to change underneath her while she is editing,
even if the underlying object in Ambassador changes, for example another user makes a change or some automated
process makes a change: in either case, we want the user experience to be smooth and predictable.

So here's how it really works:

### When the User starts Editing

(1) When the user starts an edit session on view B, view B (2) makes a copy of model B (the new copy is
model B'), and then (3) changes itself (view B) to use model B' as its model. (4) View B (now B') keeps a pointer
to the original model B for reasons that will become clear in a bit.

                       [models]               [views]
                      +---------+           +---------+
    [Ambassador]      | +-----+ |-----------| +-----+ |
      Host A          | |  A  |---------------|  A  | |
      Host B          | +-----+ |           | +-----+ |
      Host Q          | +-----+ - - - -(4)- - +-----+ |
                      | |  B  |        +-(3)--|  B' |<-----(1)
                      | +-----+ |      |    | +-----+ |
                      |         |      v    |         |
                      |         |  +-----+  |         |
                      |         | (2) B' |  |         |
                      |         |  +-----+  |         |
                      | +-----+ |           | +-----+ |
                      | |  Q  |---------------|  Q  | |
                      | +-----+ |           | +-----+ |
                      +---------+           +---------+

The user makes all the changes to view B' and thus model B' that she wants. Because model B' is not part
of the collection model, and because model B still exists in the collection model, all the 
[Ambassador --> Model](ambassador_to_model_to_view.md) change propagation continues to work, i.e., if
changes to model B come through the snapshot, the original model B gets updated, any other views are
updated, etc. However because view B' is listening to model B', view B' does not yet those updates which
is the behavior we want: the user is editing view B' so we don't want the view to surprise her by changing
"automagically": the only changes in view B' should come through her actions in the UI.

### When the User presses Save to end the Edit session

(1) When the user presses Save to end the Edit session, view B' (2) swaps its model B' (the new data)
into the model collection and (3) saves the old data model B. (4) The new model B' uses the API call
to send the new data to Ambassador. And finally, the new model B' and the view B' are
marked as "pending" (indicated by `/////`) and (5) a five second timer is started.

                       [models]               [views]
                      +---------+           +---------+
    [Ambassador]      | +-----+ |-----------| +-----+ |
      Host A          | |  A  |---------------|  A  | |
      Host B <--+     | +-----+ |           | +-----+ |
      Host Q    |     | +/////+ |           | +/////+ |
                +-(4)--(2) B' |---------------|  B' |<-----(1)
                      | +/////+ |      +- - - +/////+ |
                      |         |      |    |         |
         (5)          |         |  +-----+  |         |
       [timer]        |         | (3) B  |  |         |
                      |         |  +-----+  |         |
                      | +-----+ |           | +-----+ |
                      | |  Q  |---------------|  Q  | |
                      | +-----+ |           | +-----+ |
                      +---------+           +---------+

#### When the Snapshot has data, but not new data

(1) The snapshot data arrives every second (currently). The snapshot may continue to contain data B
even though our model has data B'. Normally if model B had data B' when the snapshot had data B, the
system would update the model to data B, but that would cause a weird UI behavior:
1. User edits B -> B', see B' on the screen
2. User press Save, still sees B' on the screen
3. Ambassador hasn't completed the save yet, so when the snapshot arrives it resets the data to B, 
so the user sees B' flip back to the original B
4. Then the save completes, so when the next snapshot arrives the data is B' and the data is set to B',
so the user sees B flip back to B' again.
5. The user is very confused.

So to prevent this, when model B' is marked as "pending", the normal snapshot updates do not update the model.

                       [models]               [views]
                      +---------+           +---------+
    [snapshot] =(1)=> | +-----+ |-----------| +-----+ |
      Host A          | |  A  |---------------|  A  | |
      Host B          | +-----+ |           | +-----+ |
      Host Q          | +/////+ |           | +/////+ |
                      | |  B' |---------------|  B' | |
                      | +/////+ |      +- - - +/////+ |
                      |         |      |    |         |
                      |         |  +-----+  |         |
                      |         |  |  B  |  |         |
                      |         |  +-----+  |         |
                      | +-----+ |           | +-----+ |
                      | |  Q  |---------------|  Q  | |
                      | +-----+ |           | +-----+ |
                      +---------+           +---------+


#### When the Snapshot indicates the Save was successful

(1) When new snapshot data arrives for B (data B'), then we know that Ambassador and Kubernetes
successfully completed the save, so model B' and view B' "pending" flags are removed and (2) the
old model B is deleted.

                       [models]               [views]
                      +---------+           +---------+
    [snapshot] =(1)=> | +-----+ |-----------| +-----+ |
      Host A          | |  A  |---------------|  A  | |
      Host B'         | +-----+ |           | +-----+ |
      Host Q          | +-----+ |           | +-----+ |
                      | |  B' |---------------|  B' | |
                      | +-----+ |           | +-----+ |
                      |         |           |         |
                      |         |    (2)    |         |
                      |         | XXX B XXX |         |
                      |         |           |         |
                      | +-----+ |           | +-----+ |
                      | |  Q  |---------------|  Q  | |
                      | +-----+ |           | +-----+ |
                      +---------+           +---------+


#### If the Timer expires meaning failed Save

(1) If the timer expires before we receive the new B' data, then we assume that something went
wrong with the save, i.e., that it has failed. And thus we (2) return the model B back to
the collection model and (3) discard the new data (model B') and (4) reset the view back to B.

    [timer]==(1)===>    [models]               [views]
                      +---------+            +---------+
    [snapshot]        | +-----+ |------------| +-----+ |
      Host A          | |  A  |----------------|  A  | |
      Host B          | +-----+ |            | +-----+ |
      Host Q          | +-----+ |            | +-----+ |
                      |(2) B  |-------(4)------|  B  | |
                      | +-----+ |            | +-----+ |
                      |         |            |         |
                      |         |    (3)     |         |
                      |         | XXX B' XXX |         |
                      |         |            |         |
                      | +-----+ |            | +-----+ |
                      | |  Q  |----------------|  Q  | |
                      | +-----+ |            | +-----+ |
                      +---------+            +---------+
                      

### When the User presses Cancel to end the Edit session

(1) If the user presses "Cancel" to cancel the Edit session rather than saving it, we 
(2) change back to listening to the original model B, (3) discard model B', and (4) reset
the view to the data in model B.

                        [models]               [views]
                      +---------+            +---------+
    [snapshot]        | +-----+ |------------| +-----+ |
      Host A          | |  A  |----------------|  A  | |
      Host B          | +-----+ |            | +-----+ |
      Host Q          | +-----+ |            | +-----+ |
                      | |  B  |------(2,4)-----|  B  |<-----(1)
                      | +-----+ |            | +-----+ |
                      |         |            |         |
                      |         |    (3)     |         |
                      |         | XXX B' XXX |         |
                      |         |            |         |
                      | +-----+ |            | +-----+ |
                      | |  Q  |----------------|  Q  | |
                      | +-----+ |            | +-----+ |
                      +---------+            +---------+



 
