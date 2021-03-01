# configmap-watcher

**This repository is being used temporarily as a solution that works with our current
fork of jetstack's cert-manager.  
An alternate solution will be used soon.  This repository will then be archived.**

This controller allows you to add an annotation to a deployment indicating the deployment
should be restarted any time a change is detected in the specified configmap.
