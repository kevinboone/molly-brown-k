# Kepler network protocol specification

Version 0.1a, May 2026

This document is placed in the public domain under the terms of the
[Creative Commons CC0 1.0 Universal Public Domain Dedication](https://creativecommons.org/publicdomain/zero/1.0/).

## Abstract

This document specifies the Kepler protocol for file transfer. It can be
thought of as an incremental improvement over Gemini, as Gemini is as an
incremental improvement over Gopher [RFC1436]. Neither protocol is intended to
be a stripped down HTTP [RFC7230]. 

Kepler runs over TCP [STD7], optionally with encryption provided by TLS
[RFC8446], using simple request/response interactions. It can serve arbitrary
content with specific MIME types [RFC2045], but is most frequently used to
serve lightweight hypertext documents of various types.

## Conventions used in this document

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD",
"SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be
interpreted as described in [BCP14]. 

## Overview

Kepler is designed to be sufficiently close to Gemini that the same server
program could handle both Gemini and Kepler. 

Like Gemini, Kepler provides a simple, request/response protocol that is easy
to implement, while still being useful. Also like Gemini, Kepler is intentionally
inflexible and difficult to extend. Unlike Gemini, however, Kepler handles content
timestamps, allowing documents to be cached after retrieval. It also allows the
server to tell the client how much data to expect in its response.

Plaintext Kepler is served on port 2009 by default (2009 is the year of the
launch of the Kepler Space Telescope). TLS-encrypted Kepler is served on
port 10009 (that is, 8000 + 2009).

When using TLS, servers and clients MUST accept both self-signed and CA-signed
certificates.  Both SHOULD, however, take steps to detect unusual use of a
certificate, such as a change in the certificate associated with a particular
host.

Addressing in Kepler is based on URIs [STD66], with the following modifications:

- the schemes used are "kepler" and "keplers", for plaintext and TLS transactions
  respectively;

- the "userinfo" portion of a URI MUST NOT be used;

- an empty path component and a path component of "/" are equivalent; 
  servers MUST support both without sending a redirection;

- the "port" component of the URI defaults to 2009 for `kepler` and 10009 for
  `keplers`;

- clients SHOULD use a hostname rather than an IP address in the authority
  section of a URI where practicable;

- clients MUST NOT use URIs containing fragments.

## Connection behaviour

A Kepler transfer takes place within a single TCP/IP session and, when using
`keplers`, a single TLS session. At most one document can be transferred in a
session, but there is no limit on the size of this document (it may even be
unbounded), nor on the time taken to transfer it.

The server MUST close its connection to the client once it has sent a
complete response. The server MUST clear down the TLS session before
closing the TCP/IP session if it is practicable to do so.

The client SHOULD read data until the server closes the connection. Even when
the client knows the amount of data the server is going to send, it SHOULD 
still allow the server to close the connection.

However, the server MUST tolerate a situation in which the client closes its
connection prematurely, whether as part of normal operation or because of
an error. 

## Requests

The client's request consists of an absolute URI followed by a single space,
then an integer 'last-cached' time, then a space, then a CR (character 13) and
LF (character 10). The augmented BNF [STD68] for this sequence is:

```
request = absolute-URI last-cached-time CRLF

	; absolute-URI      from [STD66]
        ; last-cached-time  seconds from the Unix Epoch
	; CRLF              from [STD68]
```

The server MUST reject requests where the URI 

- contains a userinfo portion; 
- contains a fragment;
- is more than 1024 bytes long.

A client SHOULD send a '/' rather than an empty path, but a server MUST treat
an empty path as if it were '/'.

`last-cached-time` MUST be a non-negative integer. A client MAY specify zero
for this value in all requests if it does not cache content.

If the last-cached-time is non-zero the server SHOULD respond with a
not-changed indication (see below) if the content has, in fact, not changed
since the specified time.

## Responses

Upon receiving a request, the server sends back a response header. In the
case of a successful request, the header is followed by the content requested
by the client. Response headers MUST be UTF-8 encoded text and MUST NOT begin
with a Byte Order Mark (BOM). A response header consists of a two-digit
status code, possibly followed by some additional information (which depends upon
the response being sent), followed by a CR and LF. 

```
reply    = input / success / redirect / tempfail / permfail / auth / unchanged

        input     = "1" DIGIT SP prompt        CRLF
        success   = "2" DIGIT SP length updated mimetype CRLF body
        redirect  = "3" DIGIT SP URI-reference CRLF
                        ; NOTE: [STD66] allows "" as a valid
                        ;       URI-reference.  This is not intended to
                        ;       be valid for cases of redirection.
        tempfail  = "4" DIGIT [SP errormsg]    CRLF
        permfail  = "5" DIGIT [SP errormsg]    CRLF
        auth      = "6" DIGIT [SP errormsg]    CRLF
        unchanged = "70" [SP errormsg]         CRLF

        prompt    = 1*(SP / VCHAR)
        mimetype  = type "/" subtype *(";" parameter)
        errormsg  = 1*(SP / VCHAR)
        body      = *OCTET

        VCHAR    =/ UTF8-2v / UTF8-3 / UTF8-4
        UTF8-2v  = %xC2 %xA0-BF UTF8-tail ; no C1 control set
                 / %xC3-DF UTF8-tail

        length   = [-]*DIGIT
        updated  = [-]*DIGIT

	; URI-reference from [STD66]
        ;
        ; length         integer
        ; last-updated   integer timestamp
	;
	; type           from [RFC2045]
	; subtype        from [RFC2045]
	; parameter      from [RFC2045]
	;
	; CRLF           from [STD68]
	; DIGIT          from [STD68]
	; SP             from [STD68]
	; VCHAR          from [STD68]
	; OCTET          from [STD68]
	; WSP            from [STD68]
	;
	; UTF8-3         from [STD63]
	; UTF8-4         from [STD63]
	; UTF8-tail      from [STD63]
```

The VCHAR rule from [STD68] is extended to include the non-control codepoints
from Unicode (and encoded as UTF-8 [STD63]).

The body of the response is generally not governed by this Specification,
because its format depends upon the type of content being served. 

However, when the MIME type of the content is any subset of "text" (including
"text/gemini" and "text/markdown"), and the content is Unicode with any
encoding, the body SHOULD NOT begin with a byte-order marker (BOM).  Instead,
the encoding should be communicated to the client through the "charset"
parameter appended to the MIME type. If a body declared to be of type
"text/gemini" or "text/markdown" begins with a BOM, clients SHOULD ignore it
when parsing the document.

After sending the complete response (which may include the requested content),
the server closes the connection. Where the communication was over TLS, the
server MUST use the TLS `close_notify` mechanism to inform the client that no
more data will be sent.

Response status code range from 10 to 79 inclusive, although not all values are
used. They are grouped such that a client can use the initial digit alone to
handle the response; the additional digit further clarifies the status, and it
is RECOMMENDED that clients use the additional digit when deciding what to do.
Servers MUST NOT send status codes that are not defined.

## Status codes

There are seven groups of status codes:

```
    10-19 Input expected
    20-29 Success
    30-39 Redirection
    40-49 Temporary failure
    50-59 Permanent failure
    60-69 Client certificates
    70-79 Cache control
```

A client MUST reject any status code less than '10' and greater than '79'.  A
client SHOULD handle undefined status codes between '10' and '79' in line with
the initial digit. For example, it should treat a status of '14' as if it
were '10';

### Status 10-19: input expected

The server needs user input. A client MUST use the text after the status code,
if there is any, to prompt the user.  If the user provides the information, the
client then makes a subsequent request for the same URI, with the user input
included as the query portion. 

```
	input  = "1" DIGIT SP prompt CRLF
	prompt = 1*(SP / VCHAR)
```

For consistency, the client MUST send the last-cached field in its new request,
but MAY set it to zero, and usually should. One reason not to set the
last-cached field to zero is that the client recognizes the user's input after
prompting, and knows that this input gave rise to an earlier response that the
client was able to cache. Most likely, however, the server will not have sent
an 'updated' time when it earlier sent the content, because content provided in
response to user input is usually dynamically-generated, and not suitable for
caching.

Spaces in the user input MUST be encoded as '%20'.  Clients SHOULD allow for
the entry of input composed of multiple lines. In such cases the line-breaks
in the user input SHOULD be encoded as '%0A'. Servers SHOULD recognise both
'%0A' and '%0D%0A' as line-breaks.
 
If a client receives a 1x response to a URI that already contains a query
string, the client MUST replace the query string with the user input when
making the new request. For
example, if this URI results in a 10 response:

```
	keplers://example.net/search?hello
```

The client will send as a request:

```
	keplers://example.net/search?the%20user%20input
```

and not, for example:

```
	keplers://example.net/search?hello&the%20user%20input
```

There are two status codes in this category.

### Status 10

Input with no special handling. The client MUST prompt a user for input
and repeat the request as outlined about.

### Status 11: sensitive input

As status 10, but the server is indicating that the input is likely to be
sensitive, such as a password.  Clients MUST present the prompt as with status
code 10, but the user's input SHOULD be concealed from view in some manner that
is appropriate for the client.  

### Status 20-29: success

The request was handled, and the server has content to send to the client. The
content will directly follow the response header.

```
success = "2" DIGIT SP length updated mimetype CRLF body
        length   = [-]*DIGIT
        updated  = [-]*DIGIT
	mimetype = type "/" subtype *(";" parameter)
	body     = *OCTET

        ; length         integer
        ; last-updated   integer timestamp
        ; length         integer
        ; last-updated   integer
```

The 'length' field of the response indicates the size in bytes of the data to
follow.  If the server does not know the length at the time of sending the
response header, it MUST send '-1'. However, in circumstances where it is
feasible for the server to determine the length, it SHOULD send the correct
value, and clients SHOULD use it to indicate to the user the progress of the
transfer. 

If the server does send the length, it MUST provide exactly that much data
to the client.

The 'updated' field indicates when the content was last updated on the server,
expressed as seconds after the Unix Epoch. The server SHOULD send an accurate
timestamp in circumstances where it can determine one, and where the request is
idempotent; but it MAY legitimately send '-1' as the 'updated' field in all
responses.  Doing this makes Kepler behave like Gemini.  

If the request is not idempotent, the server MUST send '-1', so the client knows
that the response should not be cached. 

The response body is just raw content, text or binary, as with gopher
[RFC1436].  Kepler defines no support for compression, chunking or any other
kind of content or transfer encoding. The server closes its end of the
connection after the final byte if the body -- there is no "end of response"
marker.

When the response body has one of the "text/" MIME types, clients which present
these documents to users MUST accept both CRLF and LF as line-breaks markers.

This Specification does not mandate that clients handle any particular content
type. However, it is envisaged that end-user clients will render and display
simple hypertext formats like Gemtext (`text/gemini`) and Markdown
(`text/markdown`) internally.  It is also expected that clients will display at
least some content types that are referenced by a such a document --
particularly images. 

If a client does not handle the content itself, it should allow the user 
to carry out some useful action, such as saving it to storage. 

The only status code defined under this group is 20.

### Status 30-39: redirection

These responses indicate that the client should retrieve the content from some
new location, or using a different protocol. 

```
	redirect = "3" DIGIT SP URI-reference CRLF
                        ; NOTE: RFC-3987 allows "" as a valid
                        ;       URI-reference.  This is not intended to
                        ;       be valid for cases of redirection.
```


The URI-reference can be absolute or relative. If absolute, it may indicate the
same host or a different host, using the same protocol or a different protocol.
That is, a server MAY redirect a Kepler request to a Gemini server or even an
HTTP server, although the client may not be able to handle such a request by
itself.

If a server sends a redirection in response to a request with a query string,
the client MUST NOT apply the query string to the new location. However, the
server MAY include a query string in its URI-reference. 

Clients MUST limit the number of redirections they follow to five. 

There are two defined status code in this category.

### Status 30: temporary redirection

The redirection is temporary, and the client SHOULD in future request the
content with the original URI.

### Status 31: permanent redirection

The location of the content has changed permanently. Clients SHOULD use the new
location to retrieve the content in future.

### Status 40-49: temporary failure

This status indicates that the request has failed, and there will be no response body. 
However, the server expects the failure to be short-lived, and that
the same request might succeed later.

```
        tempfail = "4" DIGIT [SP errormsg] CRLF
	errormsg = 1*(SP / VCHAR)
```

Clients SHOULD display a localized description of the failure to the user.  The
server MAY provide an optional error message in its response and, if it does,
the client SHOULD display it, or otherwise use it to assist the user.  There
are five status codes in this category.

### Status 40: unspecified condition

An unspecified error condition exists on the server that is preventing the content
from being served, but a client MAY try again at its discretion to obtain the
content. The server MUST respond with this status if none of the others in the
same category seem appropriate.

### Status 41: server unavailable

The server is temporarily unavailable due to overload or maintenance (cf HTTP
503).

### Status 42: 'CGI error'

A CGI process, or some similar system for generating dynamic content, failed
unexpectedly or timed out before producing a response.

### Status 43: proxy error

A proxy request failed because the server was unable to successfully complete a
transfer with the downstream host (cf HTTP 502, 504).

### Status 44: slow down

The server is under too much load to satisfy the client's requests at present.
The client SHOULD double its interval between requests, and continue to do so
until it receives some response other than status 44.  

### Status 50-59: permanent failure

The request has failed, and there will be no response body. The client SHOULD
NOT retry the request in the short term, because it will probably fail again. 

```
	permfail = "5" DIGIT [SP errormsg] CRLF
	errormsg = 1*(SP / VCHAR)
```

Clients SHOULD display a localized description of the failure to the user,
using whatever information the server provides, if any. The server MAY provide
an optional error message in its response and, if it does, the client SHOULD
display it, or otherwise use it to assist the user.  There are five status
codes under this category.

### Status 50

This is the general permanent failure status. The server MUST send it if
none of the other codes in this category describe the problem better.

### Status 51: not found

The requested resource could not be found at this time, and no further
information is available to explain why. The content may be available again at
some later time, but this cannot be guaranteed (cf HTTP 404).

### Status 52: gone

The requested resource is no longer available, and will not be available again.
Search engines and similar tools should remove this resource from their
indices. Content aggregators should stop requesting the resource and convey to
their users that the subscribed resource is gone (cf HTTP 410).

In practice, a server is unlikely to know whether a missing resource will
reappear at some point, unless it performs sophisticated content management.
It MAY use status 51 in all cases of missing content.

### Status 53: proxy request refused

The request was for a resource at a domain not served by the server, and the
server does not accept proxy requests. The server MUST NOT return this status
in situations where a downstream proxy refused the request: this situation is
covered by status 43. 

### Status 59: bad request

The server was unable to parse the client's request, presumably due to a
malformed request, or the request violated the constraints listed in the
Request section.

# Status 60-69: client certificate issue

These responses will only be returned when using the TLS-encrypted `keplers` protocol. 
They indicate that the client did not provide a certificate, or provided one that
was inadequate in some way.

```
	auth     = "6" DIGIT [SP errormsg] CRLF
	errormsg = 1*(SP / VCHAR)
```

Clients SHOULD use whatever information the server returns to advise the user of
the problem and the suggested action. If the server returns an error message,
the client SHOULD use it to guide the user. In general, the client SHOULD NOT
try to correct this problem itself by, for example, generating a certificate
automatically.

There are three status codes in this category.

### Status 60: certificate required

Access to the content requires a client certificate, and the client did not
provide one. The client SHOULD NOT repeat the request without a certificate. A
server MAY require a certificate only for specific paths, and it MAY ask for
a different certificate for each path. 

### Status 61: certificate not authorized

The certificate supplied by the client does not permit access to the requested
resource. The problem is not with the certificate itself, which may be
authorised for other resources. The client SHOULD NOT repeat the request
without allowing the user to try to correct the problem. 

### Status 62: certificate not valid

The supplied client certificate was not valid, regardless of the requested resource. For example, 
the current date and time is outside the certificate's validity period, or there is some
other violation of x.509 standards. Servers MUST NOT reject a certificate because it
is self-signed, but MAY reject it because of evidence of tampering. The client
MUST report this problem to the user.

## The use of TLS

### Minimum version requirements

`keplers` clients and servers MUST support TLS 1.2, and SHOULD support TLS 1.3.
Clients SHOULD warn their users against using a client certificate with a TLS
1.2 server, as the certificate will be sent in the clear. However, the client
MAY make the request if the user allows.

### Closing connections

`keplers` servers MUST use the TLS `close_notify` implementation to close the
connection, but SHOULD correctly handle a situation where the other party does
not (such as a low level socket error that closes the socket without properly
terminating the TLS connection). A client SHOULD notify the user of such an
occurrence; a server might just log it, or do nothing.

### Server Name Indication (SNI)

Clients and servers MUST support TLS SNI. Clients MUST include hostname
information when making requests for URIs where the authority section is a
hostname. If requesting a URI where the authority section is an IP address
(contrary to the recommendation in the overview), clients SHOULD omit SNI
information completely, rather than setting it to an empty value.

### TLS server certificate validation

This specification does not mandate how, or whether, clients should validate
server certificates. If practicable, however, clients SHOULD take steps,
perhaps using a trust-on-first-use approach, to detect tampering and
man-in-the-middle attacks.

### Non-equivalence of URIs

Clients MUST NOT assume that a `kepler` URI and a `keplers` URI that differ
only in the 'scheme' part pertain to the same resource even though, in practice,
they usually will. A server is not required to support both protocols and, when
it does, it MAY redirect from one to the other. Moreover, when a host supports
both `kepler` and `keplers`, it MAY use different server applications to do so.
There is no way for the server administrator to be absolutely certain that URIs
with the same path correspond to the same resource.

If a client caches responses from the server, it should maintain independent
cached versions for plaintext and encrypted versions of the same URI path,
unless it is certain (e.g., by matching checksums) that they relate to the same
document.

## Proxies

A Kepler request MAY be routed to the server through one or more
proxies. Clients that support this mode of operation SHOULD indicate to the
user that proxies are in use, if they know. Usually the client will only
know if the user has set a proxy in the client's configuration.  

Failing to support proxies does not make a client non-compliant.

A Kepler server MAY handle some requests internally, and proxy others. 

A proxy MAY make a request using a different protocol than the one the
client is using to communicate with the proxy. For example, a proxy
may accept Kepler clients, but communicate downstream using HTTP.

If a proxy is able to make a request and receive any response from the
downstream server, it should return the server's response unmodified to the
client. If the proxy is unable to make the request at all, it should respond to
the client with status code 43, as described above. In such a case, the proxy
SHOULD include an error message in its response, containing such information as
it is able to provide (e.g., connection refused downstream).

## Cache control

The ability of clients and proxies to cache content is what distinguishes
Kepler from Gemini. When a client receives a response, it MAY cache the
body -- either in memory or longer-term storage -- against the timestamp
and URI. When the user requests the same URI again, the client 
MAY put the timestamp of the cached content in its request. If the server
determines that its copy of the content has not changed since the 
client cached it, it MAY respond with status code 70, indicating to
the client that it can safely present its cached version.

In addition, since the client knows when the content was last updated
-- because the server SHOULD provide that information in 'OK' responses
-- the client MAY use heuristic methods to avoid interacting with the
server at all. For example, if the client last requested a URI five
minutes ago, and the server says that content was last updated a year ago, 
it's almost certainly safe for the client o show the user the same
content, and avoid a call on the server at all.

However, client-side caching adds considerably to the complexity of
a client. In order that Kepler be as simple to implement as Gemini,
a client MAY decline to cache at all. In that case, it would
supply '0' as its `last-cached` value in all requests, and ignore
any timestamp in the server's response.

Adopting this approach, of course, removes any advantage that Kepler
has over Gemini.

## Authentication and authorization

Like Gemini, Kepler provides no means of client authentication other than a
client's TLS certificate. It follows that plaintext Kepler provides no
authentication at all, and therefore offers no scheme for authorizing
particular clients for particular resources.

This Specification does not mandate how a Kepler server should carry out
authorization, that is, how it should relate client certificates to resources.
Authorization MAY be all-or-nothing, that is, if a client is granted access to
the server, it has access to every valid URI.

Alternatively, a server may regulate authorization to the level of individual
files, and demand a different client certificate for each file.

Such a scheme would be confusing to users, and difficult for a client developer
to implement. Therefore a server SHOULD ensure that, if a client certificate
grants access to some URI, it also grants access to its parent URIs. That is,
if a certificate provides access to

```
keplers://acme.com/foo/bar
```

it also provides access to

```
keplers://acme.com/foo/
```

so long as `foo` is a valid URI. In all other particulars, authorization is at
the discretion of the server.

A client MAY send information collected from the user in the clear (that is,
using the plaintext `kepler` protocol). However, because this a security and a
privacy hazard, the client SHOULD warn the user if it is about to do so.

## Examples

### Simple retrieval

Here the client requests a specific document, indicating (with the value "0")
that the server should return the content, regardless when it was last updated.

```
Client: [opens connection]
Client: keplers://example.net/index.md 0 CRLF
Server: 20 text/markdown 1548 1777745482 CRLF content...
Server: [closes connection]
```

The server responds with the length (1548) and the timestamp of the last
update. The client may cache this response along with the URI and timestamp
for future use.

### Permanent failure

```
Client: [opens connection]
Client: keplers://example.net/data CRLF
Server: 52 Resource deleted CRLF
Server: [closes connection]
```

The "52" response indicates that the resource once existed at the given
URI, but is no longer available. A "51" response is less final: 
although the request is unlikely to succeed if repeated in the short
term, the resource might plausibly be restored at some point.

### Retrieval with cached document

```
Client: [opens connection]
Client: keplers://example.net/index.md" 1777745482 CRLF
Server: 70 Not changed CRLF
Server: [closes connection]
```

Here the client indicates to the server that it has a cached copy of the
document with a specific timestamp. The server responds that it does not
have a more up-to-date version.

### User input 

```
Client: [opens connection]
Client: keplers://example.net/search 0 CRLF
Server: 10 Please input a search term CRLF
Server: [closes connection]
Client: [prompts user, gets input]
Client: [opens connection]
Client: keplers://example.net/search?gemini%20search%20engines 0 CRLF
Server: 20 text/markdown 1548 -1 CRLF content...
Server: [closes connection]
```

The server responds that the user should enter a search term. The client
prompts the user, then repeats the request with the user's search phrase as the
query. The client supplies '0' for the 'last-cached' field, because it doesn't
have a cached response (this will usually be the case with user input). The
server responds with '-1' as the 'updated' field because it generated the
response dynamically, and it knows that the same request may produce a
different response at another time. 

### Redirection to change protocol

```
Client: [opens connection]
Client: kepler://example.net/document.html 0 CRLF
Server: 30 keplers://example.net/document.html CRLF
Server: [closes connection]
Client: [opens connection]
Server: keplers://example.net/document.html 0 CRLF
Server: 20 mimetype 12345 1777748765 CRLF content...
Server: [closes connection]
```

The client requests a document using the plaintext `kepler` protocol.  The
server responds that it should try the request for the same resource using the
encrypted `keplers` protocol.


## References

[BCP14: Key words for use in RFCs to Indicate Requirement Levels](https://datatracker.ietf.org/doc/bcp14/)

[RFC2045: Multipurpose Internet Mail Extensions (MIME) Part One: Format of Internet Message Bodies](https://www.rfc-editor.org/rfc/rfc2045)

[RFC3987 Internationalized Resource Identifiers (IRIs)](https://www.rfc-editor.org/rfc/rfc3987)

[STD63: UTF-8, a transformation format of ISO 10646](https://datatracker.ietf.org/doc/rfc3629/)

[STD66: Uniform Resource Identifier (URI): Generic Syntax](https://datatracker.ietf.org/doc/std66/)

[STD68: Augmented BNF for Syntax Specifications: ABNF](https://datatracker.ietf.org/doc/std68/)

[STD80: ASCII format for network interchange](https://datatracker.ietf.org/doc/std80/)

[RFC1436: The Internet Gopher Protocol](https://www.rfc-editor.org/rfc/rfc1436)

[RFC7230: Hypertext Transfer Protocol](https://www.rfc-editor.org/rfc/rfc7230)

[RFC8446: The Transport Layer Security (TLS) Protocol Version 1.3](https://www.rfc-editor.org/rfc/rfc8446)

[STD7: Transmission Control Protocol](https://datatracker.ietf.org/doc/std7/)

