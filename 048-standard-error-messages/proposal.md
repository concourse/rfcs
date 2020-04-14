# Summary

This RFC proposes that the Concourse API standardize on returning human and
machine readable error messages.

Authors: @taylorsilva @aoldershaw

# Motivation

As a consumer of the Concourse API (e.g. `fly`) I want to return a helpful error
messages to my end-user. This proves to be difficult to do because all I can do
is imprecisely reason about why an error occured based on the HTTP status code
returned by the API.

Example: The API endpoint for getting a pipeline config may return HTTP 404 in
two situations: 1) the config is not found and 2) the pipeline is archived. It's
currently not possible without further introspecition to distinguish between
these two cases at the client level.


# Proposal

When the Concourse API returns an error (4xx and 5xx status codes) it should
return the relevant HTTP status code and set the body to a JSON object of the
following form:

```json
{
  "type": "MachineReadable",
  "message": "human readable"
}
```

If further arbitrary details are required any number of extra keys can be added
to the JSON object.


# Open Questions

* What format should we use? e.g. There's the [jsonAPI spec](https://jsonapi.org/format/#errors)

# Answered Questions



# New Implications

Allows for more flexible client-side error handling and more meaningful error
messages.
