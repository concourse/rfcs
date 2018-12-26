# Summary

> Separate secrets from pipeline definitions

# Motivation

> At the moment secrets and pipelines are mixed together. You can insert a secret via a ((variable)).  
> In this way you have to set and know every secret on every set-pipeline. There are ways to simplify this, like having it in a file and passing it into the set-pipeline; this makes it much more insecure, as you have to store the file somewhere and/or care for not commiting it to git.  

# Proposal

> At my point of view it would be nice to separate secrets from pipeline definitions. If there would be a way to set these separately, the following scenarios would be much easier to implement:
* Update pipelines without even having the need to know the secrets.  
This would be a big plus, for pipeline maintenance. Today people have to look up the secrets via get-pipeline and transfer it to the updated set-pipeline by hand. This is a big vector for errors.
* Setting/Updating pipelines dynamically without manual interaction.  
It would open up the door for having something like a ".travis"-file for concourse  
>
> In addition the pipelines wouldn't have to be encrypted in the database, which would enhance performance in big setups/pipelines. Most of pipeline definitions isn't secrets, nevertheless the complete data is stored encrypted at the moment.

# Open Questions

> * How could secrets be separated without implementing just another password store?  
> * Would this be in addition to current behavior or as a replacement?
