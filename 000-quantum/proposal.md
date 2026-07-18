# Summary

Introduce Concourse support for post-quantum cryptographic algorithms.

# Motivation

Post-quantum cryptography is getting more and more relevant. 

For example, Cloudflare statistics show that post-quantum cryptography is already pretty popular on the Internet (https://radar.cloudflare.com/post-quantum).

X25519MLKEM768 is a key exchange algorithm that is supported in the TLS protocol and includes a classical (elliptic curve X25519) and a post-quantum (MLKEM768) part.

It has already been implemented in major OSS projects, including Mozilla Firefox (https://www.firefox.com/en-US/firefox/132.0/releasenotes/) and OpenSSL (https://openssl-library.org/news/openssl-3.5-notes/), and by other major products such as Google Chrome.

MLKEM768 is also supported by major SSH implementations such as OpenSSH.

Providing support for post-quantum cryptography will help keeping up with this trend - which will most likely end up with discouraging or even deprecating purely classical key exchange algorithms - and increasing security, especially for people who already using post-quantum-capable HTTPS and SSH clients (Chrome, Firefox, Edge, OpenSSH 10, ...).

# Proposal

## TLS

Concourse is written in Golang, whose crypto/tls library supports X25519MLKEM768 since 1.24.

In particular (https://go.dev/doc/go1.24#cryptotlspkgcryptotls):

> The new post-quantum X25519MLKEM768 key exchange mechanism is now supported and is enabled by default when Config.CurvePreferences is nil. GODEBUG setting tlsmlkem=0 reverts the default. This can be useful when dealing with buggy TLS servers that do not handle large records correctly, causing a timeout during the handshake (see TLS post-quantum TL;DR fail).

> Support for the experimental X25519Kyber768Draft00 key exchange has been removed.

> Key exchange ordering is now handled entirely by the crypto/tls package. The order of Config.CurvePreferences is now ignored, and the contents are only used to determine which key exchanges to enable when the field is populated.

This means that:

1) if Go < 1.24 is used and crypto/tls is used, bumping the Golang version is required to enable this feature (but this shouldn't be the case: https://github.com/concourse/concourse/blob/0c4db1ec814b01c46433bb552eed0fe329bc34ee/go.mod#L3)

2) if Go >= 1.24 is used and crypto/tls is used without any Config.CurvePreferences override, the underlying TLS library already does the work for you and no changes are needed,

3) if Go >= 1.24 is used and crypto/tls is used with a custom Config.CurvePreferences setting, you should manually add X25519MLKEM768 to the list to introduce support for post-quantum key exchange.

From what I can see, the only place setting custom CurvePreferences is the DefaultTLSConfig in https://github.com/concourse/concourse/blob/master/atc/config.go#L438 which includes a specific list of key exchange algorithms.

Other pieces of code seem to simply refer to the underlying crypto/tls library. For example: https://github.com/concourse/concourse/blob/master/atc/tls/runner.go#L22

Therefore, this change should be as simple as adding a new algorithm to the CurvePreferences in DefaultTLSConfig and running tests to detect any unexpected impact.

## SSH

Likewise, crypto/ssh supports MLKEM768 for SSH key exchange.

Again, DefaultSSHConfig seems the only place where SSH key exchange algorithms are explicitly defined: https://github.com/concourse/concourse/blob/0c4db1ec814b01c46433bb552eed0fe329bc34ee/atc/config.go#L477-L478

Therefore, adding mlkem768x25519-sha256 should be enough to cover SSH use cases. 


# Open Questions

> Raise any concerns here for things you aren't sure about yet.

**Could we split TLS and SSH features in two waves?**

Sure, why not, but if I'm not missing anything major the overall change is so simple that we may as well bundle TLS and SSH "upgrades" in a single RFC.

**What about old clients/servers?**

Both TLS and SSH work by *negotiating* cryptographic capabilities. 

In other words, clients and servers tell one another the cryptographic algorithms they support and agree on a set of algorithms they both support.

Therefore, in theory there shouldn't be any issue with old clients or servers: if either side doesn't support post-quantum algorithms, it will not be forced to use them.

With that said, this is something that should be tested to avoid any surprises.
