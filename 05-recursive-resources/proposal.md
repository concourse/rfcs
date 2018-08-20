# Summary

Merging resource types into resources:

* Easily introduces visibility into resource types (storing history of versions, pinning a version, etc).

* Removes the need for duplicate code (example, supporting fly check-resource-type and fly check-resource).

* Allows for easier process for creating a new resource type (can create a pipeline that runs tests on new versions of a resource type)



# Proposal

Currently, Concourse treats resources and resource types quite similarly. This proposal is to discuss the possibility of merging them together into just resources. This will end up with resource_types being defined as a resource in the pipeline and removing the resource_types section entirely. A resource type, in result, will just become a resource that will be used by another resource as its type (which will introduce recursive resources).

The idea came about with the introduction of global resource checking which has resources and resource_types being checked on it's resource config instead of the resource or resource type itself, this allowed the flow in the code for the two types to follow incredibly similar paths (including the resource and resource_type tables in the database to be almost exact replicas, the code in the radar for scanning resources and resource types to be almost exactly the same, etc).

This will introduce several advantages along with some breaking changes. A large advantage that will come along with this merge would be that resource types will finally be visible to the user. With the current model, it is very difficult to know what is happening with the resource type configured because there is no information relayed to the user about it. There is also no saved history of versions for resource types. With this change, resource types will be treated the same way as resources in the sense that you can view the versions of the resource type, view the error that came from checking, disable a version, or check for a new version manually using fly. These can also be implemented without merging the two types together but it will result in a lot of duplicate code because the code for both paths are very similar. 

The disadvantages are that the resources group in the pipeline config might get messy with both resources and resource types being defined in it, along with the fact that resources/resource-types will be under the same namespace.
