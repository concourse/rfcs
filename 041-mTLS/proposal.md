

# Summary

There is a capability for Concourse to be implemented with a reverse proxy.  
However, we would like to harden this architecture by optionally only allowing authorised reverse proxies to access Concourse in this way using certificates to validate.  


# Motivation


Our motivation for this is as follows:

By making the connection authenticate in both directions, it reduces the risk of a reverse proxy accidentally being configured to point to the wrong Concourse instance particularly in more complex networking environments.

By only allowing connections from one source, it reduces the attack surface of the Concourse instance and effectively locks the reverse proxy to the Concourse instance.  While this could also be achieved by use of firewalling, this allows the restriction to managed within the subsystem.


# Proposal
As this is something we would like to implement, we are planning on carrying out the work to support this architecture.  What we would like is for it to become an integral part of environment to avoid our having to reapply changes in subsequent releases.

As deliverables, we would provide

+ Modified branch of Concourse server
+ Unit tests for our extension and the results
+ Documentation to support the changes



# Open Questions

The mTLS exchange is defined in https://docs.oracle.com/cd/E19798-01/821-1841/bncbt/index.html and  RFC-8705  is at proposed status.  As such, while there is a risk that the RFC may not be finalised, the mechanism has been implemented in several environments particularly in B2B.





# New Implications

This change will be implemented as a optional configuration - there will be no impact on existing users should they chose not to utilise it