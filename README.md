[comment]: # ( Copyright Contributors to the Open Cluster Management project )

# configmap-watcher

**This repository is being used temporarily as a solution that works with our current
fork of jetstack's cert-manager.  
When the alternate solution is implemnted, this repository will then be archived.**

This controller allows you to add an annotation to a deployment indicating the deployment
should be restarted any time a change is detected in the specified configmap.
