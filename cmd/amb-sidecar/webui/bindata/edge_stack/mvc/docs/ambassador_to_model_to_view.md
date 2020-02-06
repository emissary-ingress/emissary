# Internals: Ambassador --> Model --> View 

Currently, the Edge Policy Console UI polls the `amb-sidecar` process in the Ambassador Edge Stack installation.
In the future we will switch to a push model via websockets or long-polling, but right now the UI polls the
AES process once a second. The AES process generates a snapshot of state data and ships it back in a JSON blob.
Each `ResourceCollection` class subscribes to the snapshot service and thus receives the entire snapshot 
each second via its `onSnapshotChange(snapshot)` method.

### Snapshot, Models, Views
The snapshot data is used to create and update the models. The models notify the views (and 
the views are linked back to the models). Not the collection-model to collection-view link as
well as the single-model to single-view links for the A, B, and Q models to the A, B, and Q views.

                       [models]         [views]
                      +---------+     +---------+
    [snapshot] =====> | +-----+ |-----| +-----+ |
      Host A          | |  A  |---------|  A  | |
      Host B          | +-----+ |     | +-----+ |
      Host Q          | +-----+ |     | +-----+ |
                      | |  B  |---------|  B  | |
                      | +-----+ |     | +-----+ |
                      | +-----+ |     | +-----+ |
                      | |  Q  |---------|  Q  | |
                      | +-----+ |     | +-----+ |
                      +---------+     +---------+

### New Data in the Snapshot Adds a Model and a View

(1) The snapshot has a new Host "S".
(2) The collection model receives the snapshot via `onSnapshotChanged()`.

                       [models]         [views]
                      +---------+     +---------+
    [snapshot] =(2)=> | +-----+ |-----| +-----+ |
      Host A          | |  A  |---------|  A  | |
      Host B          | +-----+ |     | +-----+ |
      Host Q          | +-----+ |     | +-----+ |
      (1) Host S      | |  B  |---------|  B  | |
                      | +-----+ |     | +-----+ |
                      | +-----+ |     | +-----+ |
                      | |  Q  |---------|  Q  | |
                      | +-----+ |     | +-----+ |
                      +---------+     +---------+

(3) The collection model determines that it has models for A, B, and Q so no change there. But it does
not have a model for S, so it creates a new model instance. 
(4) The collection model notifies its listeners that a new model has been created via `notifyListenersCreated()`.

                       [models]         [views]
                      +---------+     +---------+
    [snapshot] =(2)=> | +-----+ |-(4)-| +-----+ |
      Host A          | |  A  |---------|  A  | |
      Host B          | +-----+ |     | +-----+ |
      Host Q          | +-----+ |     | +-----+ |
      (1) Host S      | |  B  |---------|  B  | |
                      | +-----+ |     | +-----+ |
                      | +-----+ |     | +-----+ |
                      | |  Q  |---------|  Q  | |
                      | +-----+ |     | +-----+ |
                     (3)+-----+ |     |         |
                      | |  S  | |     |         |
                      | +-----+ |     |         |
                      +---------+     +---------+

(5) The collection view listens to the collection model and receives that "new model created" message 
(via `onModelNotification()`), so it creates a new view and (6) connects that view to the new model.

                       [models]         [views]
                      +---------+     +---------+
    [snapshot] =(2)=> | +-----+ |-(4)-| +-----+ |
      Host A          | |  A  |---------|  A  | |
      Host B          | +-----+ |     | +-----+ |
      Host Q          | +-----+ |     | +-----+ |
      (1) Host S      | |  B  |---------|  B  | |
                      | +-----+ |     | +-----+ |
                      | +-----+ |     | +-----+ |
                      | |  Q  |---------|  Q  | |
                      | +-----+ |     | +-----+ |
                     (3)+-----+ |    (5)+-----+ |
                      | |  S  |---(6)---|  S  | |
                      | +-----+ |     | +-----+ |
                      +---------+     +---------+

### Missing Data in the Snapshot Removes a Model and a View

(1) The snapshot no longer has host Q.
(2) The collection model receives the snapshot via `onSnapshotChanged()`.

                       [models]         [views]
                      +---------+     +---------+
    [snapshot] =(2)=> | +-----+ |-----| +-----+ |
      Host A          | |  A  |---------|  A  | |
      Host B          | +-----+ |     | +-----+ |
        (1)           | +-----+ |     | +-----+ |
                      | |  B  |---------|  B  | |
                      | +-----+ |     | +-----+ |
                      | +-----+ |     | +-----+ |
                      | |  Q  |---------|  Q  | |
                      | +-----+ |     | +-----+ |
                      +---------+     +---------+

(3) The collection model determines that it has a models for Q but there is no data for Q,
so it wants to delete model Q. 
(4) First the collection model notifies its listeners that model Q will be deleted via `notifyListenersDeleted()`.

                       [models]         [views]
                      +---------+     +---------+
    [snapshot] =(2)=> | +-----+ |-(4)-| +-----+ |
      Host A          | |  A  |---------|  A  | |
      Host B          | +-----+ |     | +-----+ |
       (1)            | +-----+ |     | +-----+ |
                      | |  B  |---------|  B  | |
                      | +-----+ |     | +-----+ |
                     (3)+-----+ |     | +-----+ |
                      | |  Q  |---------|  Q  | |
                      | +-----+ |     | +-----+ |
                      +---------+     +---------+

(5) The collection view listens to the collection model and receives that "model Q deleted" message 
(via `onModelNotification()`), so it delete the view for Q.

                       [models]         [views]
                      +---------+     +---------+
    [snapshot] =(2)=> | +-----+ |-(4)-| +-----+ |
      Host A          | |  A  |---------|  A  | |
      Host B          | +-----+ |     | +-----+ |
       (1)            | +-----+ |     | +-----+ |
                      | |  B  |---------|  B  | |
                      | +-----+ |     | +-----+ |
                      | +-----+ |     |         |
                      | |  Q  | |     |   (5)   |
                      | +-----+ |     |         |
                      +---------+     +---------+

(6) And then collection model deletes the model Q.

                       [models]         [views]
                      +---------+     +---------+
    [snapshot] =(2)=> | +-----+ |-(4)-| +-----+ |
      Host A          | |  A  |---------|  A  | |
      Host B          | +-----+ |     | +-----+ |
       (1)            | +-----+ |     | +-----+ |
                      | |  B  |---------|  B  | |
                      | +-----+ |     | +-----+ |
                      |         |     |         |
                      |   (6)   |     |   (5)   |
                      |         |     |         |
                      +---------+     +---------+

### Changed Data in the Snapshot Updates a Model and a View

(1) The snapshot has a new data for Host "Q".
(2) The collection model receives the snapshot via `onSnapshotChanged()`.

                       [models]         [views]
                      +---------+     +---------+
    [snapshot] =(2)=> | +-----+ |-----| +-----+ |
      Host A          | |  A  |---------|  A  | |
      Host B          | +-----+ |     | +-----+ |
    (1)Host Q'        | +-----+ |     | +-----+ |
                      | |  B  |---------|  B  | |
                      | +-----+ |     | +-----+ |
                      | +-----+ |     | +-----+ |
                      | |  Q  |---------|  Q  | |
                      | +-----+ |     | +-----+ |
                      +---------+     +---------+

(3) The collection model determines that it has a models for Q so it tells that model to update
its data from the snapshot (updates to Q'). 
(4) The Q model notifies its listeners that it has changed via `notifyListenersCreated()`.

                       [models]         [views]
                      +---------+     +---------+
    [snapshot] =(2)=> | +-----+ |-----| +-----+ |
      Host A          | |  A  |---------|  A  | |
      Host B          | +-----+ |     | +-----+ |
    (1)Host Q'        | +-----+ |     | +-----+ |
                      | |  B  |---------|  B  | |
                      | +-----+ |     | +-----+ |
                      | +-----+ |     | +-----+ |
                      |(3) Q' |---(4)---|  Q  | |
                      | +-----+ |     | +-----+ |
                      +---------+     +---------+

(5) The view for Q listens to the Q model and receives that "model updated" message 
(via `onModelNotification()`), so it updates itself and redisplays as necessary.

                       [models]         [views]
                      +---------+     +---------+
    [snapshot] =(2)=> | +-----+ |-----| +-----+ |
      Host A          | |  A  |---------|  A  | |
      Host B          | +-----+ |     | +-----+ |
    (1)Host Q'        | +-----+ |     | +-----+ |
                      | |  B  |---------|  B  | |
                      | +-----+ |     | +-----+ |
                      | +-----+ |     | +-----+ |
                      |(3) Q' |---(4)--(5) Q' | |
                      | +-----+ |     | +-----+ |
                      +---------+     +---------+
