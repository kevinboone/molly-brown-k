# Kepler support in molly-brown-k

## Scope of support

`molly-brown-k` supports the `keplers` (encrypted Kepler) and `kepler`
(plaintext Kepler) protocol, as well as the original Gemini. Sadly, it doesn't
support all three protocols at the same time: you'll need to run separate
instances with their own configuration files (see the `samples/` directory)
if you want to support all three protocols on the same server.

`molly-brown-k` interprets the `last_cached` time sent by the client, as
defined in the Kepler Specification. It returns the Kepler-specific "70" status
code if the requested file has not changed since the `last_cached` time.  The
server also returns the `length` and `updated` fields of the "20" response as
mandated in the Specification.

`molly-brown-k` accepts the clients `language` code, but only to pass it
through to CGI scripts. It doesn't yet localize error messages, for example.
The language is passed as the environment variable `LANGUAGE`.

## CGI/SCGI support 

A CGI/SCGI script that succeeds must return a complete Kepler "20" response if
the request is from a Kepler client, but a Gemini "20" response to a Gemini
client.  The Kepler response need not indicate the content-length and last
update timestamp if these are awkward to determine (or meaningless, as the
timestamp probably will be). In such a case the script should respond along the
lines:

    20 -1 -1 text/markdown

to a Kepler client, rather than

    20 text/markdown

as it must for Gemini.

To make it feasible to use the same scripts for both Gemini and Kepler
protocols, molly-brown sets the environment variable `SERVER_PROTOCOL` to
`KEPLER`, rather than `GEMINI`, for Kepler requests.

According to the Kepler Specification, the server must return a
70 (not changed) response if the request's `last_cached` time
is positive. It will usually be difficult to implement this in a CGI script,
because the "age" of the document the client received is impossible to
assess. The script should probably just return its usual output.

