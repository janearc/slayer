# Preserve the first installed result

Immutable result publication needs an atomic absent-destination operation.
Checking a path and then calling the mutable Rename method permits a
competing writer to replace the first result. The private Darwin adapter
will supply Install for a later immutable publisher.

Install receives exact source and destination paths. Its caller owns a
completed, closed temporary beside the selected destination. It calls
Darwin's exclusive rename once, retaining the native error unchanged.

Success consumes the source name and installs that inode at the absent
destination. An existing entry refuses, including directories and dangling
symlinks. There is no stat-before-rename, replacing fallback, copy or
destination removal. Unsupported exclusion remains a refusal.

The current mutable publisher keeps ordinary Rename. This new operation
does not verify content or provenance, select or protect an installation
root, flush storage, grant admission or make existing results reusable.

Errors generally require namespace reconciliation; selected documented
pre-change refusals in tests do not establish universal rollback safety.

The operation uses x/sys/unix.RenamexNp with its exclusive flag, already
available in Slayer's pinned v0.49.0 dependency. Apple's pinned manual
describes existing-entry refusal and unsupported-filesystem refusal:
[Darwin rename manual](https://raw.githubusercontent.com/apple-oss-distributions/xnu/xnu-12377.1.9/bsd/man/man2/rename.2).

## Paired native witnesses

Tiny fixtures in a private temporary directory exercise the real syscall:

1. First install moves the source inode and complete bytes, preserves an
   existing source reader and sibling, and leaves no source name.
   Its anti-test keeps an existing authored destination and source intact.
2. Two writers released from one start gate produce exactly one winner.
   The destination is the winner's inode and bytes; the loser keeps its
   source and receives the native existing-entry cause. The anti-test
   starts with an incumbent, so neither competing writer may win.
3. Refusal against directory and symlink entries preserves their exact
   identities, targets and child bytes. The paired malformed/missing-path
   tests retain wrapper-invalid and native not-found causes without
   installing a name.

The NUL refusal occurs during x/sys string conversion, before a syscall.

Race workers report through a buffered channel and never call testing
methods. A bounded receive prevents a lost result from hanging the suite.

No external process, render, privilege-dependent permission test, storage
flush or volume-wide experiment is needed. Results establish behavior on
the fixture volume only. The first implementation must follow actual
compile and assertion-red evidence for an explicit unsupported stub.

At stub 4700189, the package compiled with no tests run, then all eight
named tests rejected the unsupported body through assertions. Both Rapid
failures shrank to empty source and incumbent files; their exact failure
records remain in testdata.

The body now delegates the native call once. At f693d49, all 44 publication
tests passed, all four Rapid properties passed 100 cases, and every
production statement was covered. A first source review accepted the candidate;
final independent acceptance remains owed.

A local deliberate fault removed only the exclusive flag. Five of the
eight tests rejected it, including both competing-writer witnesses and
the retained incumbent byte counterexample. The good body was restored.

Two Rapid properties draw independent source and incumbent byte sequences
within the existing authored publication bounds. They check whole-byte
installation and preservation on refusal, including empty files. Every
trial has its own private directory and checked cleanup.

The native boundary itself has no injected mock: these witnesses need
the actual kernel's exclusion. A future composing publisher will use
generated mocks.
