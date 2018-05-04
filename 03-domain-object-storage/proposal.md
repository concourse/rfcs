# Summary
## DB and ATC domain objects
Domain model objects _often/always_ have both an `ATC` and a `DB` equivalent. These different equivalents live inside different packages, `atc` and `atc.db` respectively. When working with the objects in the API and their JSON representation we refer to the `ATC` object, when working with the database we use the `DB` equivalent of the object. We have functions that facilitate the conversion between `DB` and `ATC` equivalents of objects.
## DB domain objects structs and interfaces
In Concourse we declare most `DB` domain objects as both structs and interfaces with the interfaces providing a wrapper around the struct elements complimented with interface methods for functionality related to the struct.

Structs tend to be declared starting with a lowercase letter which makes the struct private to the package, the related interface usually has the same name starting with an uppercase letter making it a public interface.
Structs in the `DB` package tend to closely mirror the equivalent structs in the ATC package.

Declaration of those interfaces and structs is usually done in the same Go file where the implementation of the interface is provided.
In behavior the `DB` domain objects are primarily repository objects that are responsible for CRUD operations related to the domain model object.
## Naming
Eleven of the routines involved in the interaction with the domain model repository (our Postgres DB) have the word `factory` in their name.

Factories are usually objects that are concerned with the construction of complex/compound objects so that consumers of those objects don’t have to deal with the complexity of that construction.

In our case the `factory` Go files are responsible for interacting with the repository to perform CRUD operations.
## Idiomatic Go
In order to make our code accessible it is in our best interest to follow practices in the organization of our code that are considered native to the Go community.

_Some idiomatic Go rules in no particular order;_
* Interfaces should be defined by the consumers of the interface
* Interfaces should be as small and generic as possible
* Keep function names short, simple and meaningful
* Keep package names short, simple and meaningful and make it one lowercase word without dashes, underscores etc.
* Function and structure names should not repeat package names

# Proposal
## Domain and ATC domain objects
Given the similarity between the structs of the `DB` and `ATC` domain objects it seems natural to drop the structs that we declare for the DB objects.

We should not declare private elements on our structs unless we really want them to be private to the package. Since a large part of the interface that we use to wrap our private structs simply returns the struct values without modification it seems that we can use the public properties of the `ATC` structs instead and separate out that part of the interface that defines behavior.
## DB domain objects structs and interfaces
The interface methods that are declared in the DB package that describe behavior of the repository objects should be declared outside the `DB` package in de consuming packages of that particular behavior. That way the consumer defines the expected behavior and the fakes that we generate for testing will reside in the package of the consumer.

This will accommodate a cleaner separation of packages at development time with the consuming package not depending on the provider package to be there in any form for the creation of unit tests. It will also make it easier to provide alternate implementations for an interface if this is desirable.

## Naming
The factory objects that we have in our `DB` package generally are better described as repository objects and should be renamed to avoid confusion about what they do.

As for the other repository objects that are now named after their domain model cousins; we can leave them as they are if we’re happy with the fact that the `DB` package name sufficiently indicates that this is a repository object. If we think this can be improved upon by either renaming the package or the objects inside we can also take this approach.
# Clarification of terminology
* Idiomatic is described as “`using, denoting or containing expressions that are natural to a native speaker`”, or when we talk about programming languages, a native programmer.

# New implications and caveats
* If there are any differences between the `ATC` and `DB` domain objects we can change the `ATC` domain objects to contain the appropriate information.
* Where there are multiple consumers of an interface the right location of an interface may take some effort to find. The core Go language has several examples of this.
* A spike with a single domain object will be done to explore the side effects of the unification of `DB` and `ATC` domain objects to fully appreciate challenges of the proposed changes.
