# About the 'Kepler' protocol

This document introduces the Kepler protocol, and outlines its minor
differences from Gemini and Spartan.

## What is Kepler?

Kepler is a lightweight document retrieval protocol with almost exactly the
same syntax and semantics as [Gemini](https://geminiprotocol.net/).  

Kepler adds just enough extra information to Gemini requests and responses to
enable caching of documents, and to provide the client with guidance on the
size of the document to expect from the server. Clients are not required to
make use of the additional features; one that does not can be implemented with
exactly the same level of difficulty as a client for Gemini or Spartan.

The rationale behind Kepler is that Gemini, while ideal for small-scale usage
models, will not scale adequately to thousands of active capsules, or hundreds
of concurrent client requests. Allowing for caching has the potential to
relieve the load on servers, improve the user experience, and enable more
frequent crawling by search indexers.

Kepler may be encrypted using TLS as Gemini is, but this is not mandatory.
Different URL schemes are used to distinguish plaintext and encrypted
conversations: `kepler` and `keplers` respectively. Spartan is also a
plaintext protocol similar to Gemini, but simpler in some ways. 
Plaintext Kepler is closer to Gemini, albeit without TLS, than it is
to Spartan.

Kepler clients and servers are not compatible with Gemini clients and servers,
but the protocols are so similar that adding `keplers` support to existing
Gemini servers should be trivial. Adding basic support to existing clients
should also be trivial, if no new functionality is to be provided. 

The Kepler protocol is agnostic about the format of documents it transfers.
The response indicates the MIME type, as Gemini does. Clients need not support
anything more that Gemtext. However, it is envisaged that clients that make use
of Kepler's extended functionality will also be able to handle more
sophisticated formats like Markdown.

## Summary of the differences between Kepler and Gemini

Functional differences:

- TLS encryption is optional for Kepler
- Kepler requests carry one numeric field -- the last-cached time -- 
  that Gemini requests do not
- Kepler type-20 responses carry two additional numeric fields: a content length
  and a last-updated time
- Kepler introduces a new response code (70), to indicate that the requested
  resource is unchanged since the client last cached it

Besides the new features, there are other, minor differences between the Kepler
and Gemini specifications.

- The Kepler specification clarifies that type-30 (redirection) responses can
  be used to redirect between protocols (e.g, `kepler` to `keplers`)
- Kepler makes no claims about how much code it should take
  to implement a client, but retains the qualitative description "easy to implement"
- Kepler specifies that clients and servers must be willing to accept both
  self-signed and CA-signed certificates. This point is "under discussion" in the 
  Gemini specification
- The Kepler specification removes the Gemini Specification's details of the
  scope of client certificates and how the client should generate one. While
  this is relevant to software development, it doesn't need to be part of the
  protocol specification
- The Kepler specification simplifies the "proxies" section, removing the
  explanations and leaving just the requirements
- The Kepler specification adds new, brief sections on cache control and
  authentication/authorization. These are intended to clarify existing
  provisions, not to introduce new ones
- The Kepler specification clarifies that a request URI cannot contain
  fragments. The Gemini specification contradicts itself in this area
- The Kepler specification clarifies that the server should handle a situation
  where the client closes its connection prematurely, even though this may
  not be desirable

## URL format 

Kepler URLs have a scheme name of 'kepler' for plaintext communication, and
`keplers` for encrypted. Otherwise, they are identical to Gemini URLs, with the
same length restrictions. 

## Default ports

The default port for plaintext `kepler` is 2009 -- the year of launch of the
Kepler Space Telescope. For `keplers` it is 10009 (8000 + 2009). 

## Request line

The request line sent by the client is similar to Gemini, except:

- The URL will have scheme 'kepler' or 'keplers', rather than 'gemini', and
- after the URL, the client sends a numeric 'last-cached' time. The time format
  is seconds after the Unix epoch, so it will always be a non-negative number. If
  the client does not cache, it is reasonable for it to send "0" for this value
  in all requests.

For example;

    kepler://larsthebear.me/ 1777745482 \r\n

If the remote resource has not been updated since the last-cached time, the
server sends a 'not-modified' (70) response code, signalling that the
client can safely use its cached copy.

The client, however, is not required to maintain a cache and, even if it
does, it is not required to cache any particular document. If the client 
does not cache, then Kepler is functionally almost identical to Gemini.

## Responses

Kepler adds one new status code, 70, clarifies the meaning of code 30
(redirection), and modifies the 20 (OK) response.

### 20 response

The header line of the 20 (OK) response for the Gemini protocol is

    20 content-type\r\n

The Kepler protocol modifies this to:

    20 content-length last-updated content-type

Clients can't rely on either the `content-length` or `last-updated` fields to
be anything other than "-1" (unknown), although these fields should be present
in all responses. Of course, it is envisaged that a Kepler server _will_
provide meaningful values for these fields unless they are genuinely unknown
(for example, the response is an unbounded data stream), or the client really
should not cache (for example, the response is non-idempotent).

### 3x response

In addition to the usual redirection responses, a Kepler server may respond
with a redirection from a `kepler` URL to a `keplers` URL, or even a different
kind of protocol (e.g., HTTP) on a different server. It isn't entirely clear in
the Gemini specification exactly what the scope of redirection is.

### 70 response

If the client's request contains a non-zero last-cached time, and this time is
more recent than the server's copy of the resource, the server sends a 70
response. The response MAY contain an information message, but may equally just
consist of

    70\r\n

## Summary

It is envisaged that modifying an existing Gemini server to support Kepler
protocols will be almost a trivial operation, as the protocols are so similar.
A Kepler server will need to do some additional checks, to determine whether it
needs to respond with a document or a "not changed" status, and it will need to
add a file length and last-updated time to the "20" response.

Modifying a Gemini client to use Kepler will also be trivial -- if the client
does not use the new facilities that Kepler offers. Of course, there's little
purpose to such an approach, and implementing client-side caching could add
significantly to the complexity. However, if a client just wants to be
compatible with Kepler, rather than making full use of it, the necessary
changes are minimal.

