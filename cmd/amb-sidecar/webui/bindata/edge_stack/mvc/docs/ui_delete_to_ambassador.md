# UI delete View --> Model --> Ambassador

(1) When the user uses the UI to delete an item (via `onDeleteButton()`), the simple explanation
is that (2) the to-be-deleted model X uses the API to delete itself from Ambassador,
and then (3) the model X is deleted from the collection model which notifies its listening view X,
which is (4) then deleted from the collection view.

                       [models]         [views]
                      +---------+     +---------+
    [Ambassador]      | +-----+ |-----| +-----+ |
      Host A          | |  A  |---------|  A  | |
      Host B          | +-----+ |     | +-----+ |
      Host X          | +-----+ |     | +-----+ |
                      | |  B  |---------|  B  | |
                      | +-----+ |     | +-----+ |
                      | +-----+ |     | +-----+ |
                      | |  X  |---------|  X  |<-----(1)
                      | +-----+ |     | +-----+ |
                      +---------+     +---------+

                       [models]         [views]
                      +---------+     +---------+
    [Ambassador]      | +-----+ |-----| +-----+ |
      Host A          | |  A  |---------|  A  | |
      Host B          | +-----+ |     | +-----+ |
        /X/  <--+     | +-----+ |     | +-----+ |
                |     | |  B  |---------|  B  | |
                |     | +-----+ |     | +-----+ |
                |     |         |     |         |
                +-(2)--(3)      |     |(4)     <-----(1)
                      |         |     |         |
                      +---------+     +---------+

The reality is more complicated because the API call to Ambassador is asynchronous with the UI and relatively
slow from the point of view of the user. Thus we need to show the user a "pending" state while that API is
in process, and we need to handle both happy and sad path results from the asynchronous API call.

So here's how it really works:


### When the User presses Delete

(1) When the user presses the Delete button, the view X handles the button press by
(2) sending `doDelete()` to the model X which then (3) makes the asynchronous API call to 
Ambassador to do the delete. The model X and view X are marked as "pending" and (4) a
five second timer is started.

                       [models]         [views]
                      +---------+     +---------+
    [Ambassador]      | +-----+ |-----| +-----+ |
      Host A          | |  A  |---------|  A  | |
      Host B          | +-----+ |     | +-----+ |
      Host X <--+     | +-----+ |     | +-----+ |
                |     | |  B  |---------|  B  | |
                |     | +-----+ |     | +-----+ |
                |     | +-----+ |     | +-----+ |
                +-(3)--(2) X  |---------|  X  |<-----(1)
                      | +-----+ |     | +-----+ |
                      +---------+     +---------+

                       [models]         [views]
                      +---------+     +---------+
    [Ambassador]      | +-----+ |-----| +-----+ |
      Host A          | |  A  |---------|  A  | |
      Host B          | +-----+ |     | +-----+ |
      Host X <--+     | +-----+ |     | +-----+ |
                |     | |  B  |---------|  B  | |
                |     | +-----+ |     | +-----+ |
                |     | +/////+ |     | +/////+ |
                +-(3)--(2) X  |---------|  X  |<-----(1)
       (4)            | +/////+ |     | +/////+ |
     [timer]          +---------+     +---------+


#### When the Snapshot has data, but not deleted yet

The snapshot data arrives every second (currently). The snapshot may continue to contain new
data for X even though we are trying to delete X: for example, a background Kubernetes process might
be updating fields of X. However, while model X is marked as "pending", 
the normal snapshot updates do not update the model. 

_Note: this is a bug: we actually should 
update model X and, via notifications, view X even when pending because if the timer expires and we
return X to "real" (see below), then we would want X to have the correct real values._


#### When the Snapshot has data showing the delete happened

(1) When the snapshot data arrives without Host X, we know that the delete succeeded. Thus
we (2) delete the model X which the notifies its listeners that it is deleted, one of which 
is (3) the view X which then deletes itself. 

                       [models]         [views]
                      +---------+     +---------+
    [snapshot] =(1)=> | +-----+ |-----| +-----+ |
      Host A          | |  A  |---------|  A  | |
      Host B          | +-----+ |     | +-----+ |
                      | +-----+ |     | +-----+ |
                      | |  B  |---------|  B  | |
                      | +-----+ |     | +-----+ |
                      |         |     |         |
                      |(2) /X/  |     |(3) /X/  |
                      |         |     |         |
                      +---------+     +---------+


#### If the Timer expires meaning failed Delete

(1) If the timer expires before we receive a snapshot without the Host X, then we assume that something went
wrong with the delete, i.e., that it has failed. And thus we clear the pending flags on model X and view X.

    [timer]==(1)===>    [models]        [views]
                      +---------+     +---------+
    [snapshot]        | +-----+ |-----| +-----+ |
      Host A          | |  A  |---------|  A  | |
      Host B          | +-----+ |     | +-----+ |
      Host X          | +-----+ |     | +-----+ |
                      | |  B  |---------|  B  | |
                      | +-----+ |     | +-----+ |
                      | +-----+ |     | +-----+ |
                      | |  X  |---------|  X  | |
                      | +-----+ |     | +-----+ |
                      +---------+     +---------+
