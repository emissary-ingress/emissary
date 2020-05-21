# Edgectl Upgrade: upgrade to Ambassador API Gateway has failed

You’re running a previous version of the Ambassador API Gateway, but a more recent version exists.
To upgrade to the Ambassador Edge Stack you should have the latest version installed.
Update your installation to Ambassador API Gateway first and rerun the upgrader.

## What's next?

* Perhaps your installation has not been upgraded by the Operator yet. Try to edit the installation
  resource with `kubectl edit -n ambassador ambassadorinstallations ambassador` and remove
  the `updateWindow`.

* Reach out for help in our [Slack](http://d6e.co/slack).
