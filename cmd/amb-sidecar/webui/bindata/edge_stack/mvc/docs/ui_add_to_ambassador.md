# UI add View --> Model --> Ambassador

(1) When the user uses the UI to add a new item (via `onAddButton()`), the simple explanation
is that (2) the collection view creates a new model X' and (3) a new view X' and then (4) adds the new
model to Ambassador via an API call.

                       [models]         [views]
                      +---------+     +---------+
    [Ambassador]      | +-----+ |-----| +-----+ |<-----(1)
      Host A          | |  A  |---------|  A  | |
      Host B          | +-----+ |     | +-----+ |
      Host X'<--+     | +-----+ |     | +-----+ |
                |     | |  B  |---------|  B  | |
                |     | +-----+ |     | +-----+ |
                |     | +-----+ |     | +-----+ |
                +-(4)--(2) X' |--------(3) X' | |
                      | +-----+ |     | +-----+ |
                      +---------+     +---------+

The reality is more complicated because the API call to Ambassador is asynchronous with the UI and relatively
slow from the point of view of the user. Thus we need to show the user a "pending" state while that API is
in process, and we need to handle both happy and sad path results from the asynchronous API call.

So here's how it really works:

### When the User presses Add

(1) When the user presses the Add button, the collection view handles the button press by
(2) creating a new model X' and a (3) new view X'. The new view X' is added to the collection view
so that it gets displayed, but the new model X' is _not_ added to the collection model. This
is similar to the [edit workflow](ui_update_to_ambassador.md), but without the existing model.

                       [models]               [views]
                      +---------+           +---------+
    [Ambassador]      | +-----+ |-----------| +-----+ |<-----(1)
      Host A          | |  A  |---------------|  A  | |
      Host B          | +-----+ |           | +-----+ |
                      | +-----+ |           | +-----+ |
                      | |  B  |---------------|  B  | |
                      | +-----+ |           | +-----+ |
                      |         |           | +-----+ |
                      |         |      +-----(3) X' | |
                      |         |      |    | +-----+ |
                      |         |      v    |         |
                      |         |  +-----+  |         |
                      |         | (2) X' |  |         |
                      |         |  +-----+  |         |
                      +---------+           +---------+

The user makes all the changes to view X' and thus model X' that she wants. Because model X' is not part
of the collection model, and because model X' does not yet exists in Ambassador, all the 
[Ambassador --> Model](ambassador_to_model_to_view.md) change propagation continues to work, i.e., if
changes to model B come through the snapshot, the model B gets updated, any other views are
updated, etc.

### When the User presses Save to end the Add session

(1) When the user presses Save to end the Add session, view X' (2) adds its model X' 
into the model collection and (3) uses the API call
to send the new data to Ambassador. And finally, the new model X' and the view X' are
marked as "pending" (indicated by `/////`) and (4) a five second timer is started.

                       [models]               [views]
                      +---------+           +---------+
    [Ambassador]      | +-----+ |-----------| +-----+ |
      Host A          | |  A  |---------------|  A  | |
      Host B          | +-----+ |           | +-----+ |
      Host X'<--+     | +-----+ |           | +-----+ |
                |     | |  B  |---------------|  B  | |
                |     | +-----+ |           | +-----+ |
                |     | +/////+ |           | +/////+ |
                +-(3)--(2) X' |---------------|  X' |<-----(1)
         (4)          | +/////+ |           | +/////+ |
       [timer]        +---------+           +---------+


#### When the Snapshot indicates the Add was successful

(1) When new snapshot data arrives for X', then we know that Ambassador and Kubernetes
successfully completed the add, so model X' and view X' "pending" flags are removed.

                       [models]               [views]
                      +---------+           +---------+
    [snapshot] =(1)=> | +-----+ |-----------| +-----+ |
      Host A          | |  A  |---------------|  A  | |
      Host B          | +-----+ |           | +-----+ |
      Host X'         | +-----+ |           | +-----+ |
                      | |  B  |---------------|  B  | |
                      | +-----+ |           | +-----+ |
                      | +-----+ |           | +-----+ |
                      | |  X' |---------------|  X' | |
                      | +-----+ |           | +-----+ |
                      +---------+           +---------+


#### If the Timer expires meaning failed Add

(1) If the timer expires before we receive the new X' data, then we assume that something went
wrong with the add, i.e., that it has failed. And thus we (2) remove the new model X' from the collection
model, which notifies the new view X' which (3) removes the view X' from the collection view.

    [timer]==(1)===>    [models]               [views]
                      +---------+            +---------+
    [snapshot]        | +-----+ |------------| +-----+ |
      Host A          | |  A  |----------------|  A  | |
      Host B          | +-----+ |            | +-----+ |
                      | +-----+ |            | +-----+ |
                      | |  B  |----------------|  B  | |
                      | +-----+ |            | +-----+ |
                      |         |            |         |
                      |(2)      |            |(3)      |
                      |         |            |         |
                      +---------+            +---------+
                      

### When the User presses Cancel to end the Add session

(1) If the user presses "Cancel" to cancel the Add session rather than saving it, we 
do the same steps as the timer expiration above: (2) remove the new model X' from the collection
model, which notifies the new view X' which (3) removes the view X' from the collection view.

                        [models]               [views]
                      +---------+            +---------+
    [snapshot]        | +-----+ |------------| +-----+ |
      Host A          | |  A  |----------------|  A  | |
      Host B          | +-----+ |            | +-----+ |
                      | +-----+ |            | +-----+ |
                      | |  B  |----------------|  B  | |
                      | +-----+ |            | +-----+ |
                      |         |            |         |
                      |(2)      |            |(3)     <-----(1)
                      |         |            |         |
                      +---------+            +---------+



 
