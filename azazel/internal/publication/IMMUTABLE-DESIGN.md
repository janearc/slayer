# Publish complete bytes without replacing a result

An immutable attempt must install complete bytes at an absent name. The
existing snapshot publisher deliberately replaces its destination. This
separate composing operation uses the tested exclusive Install boundary.

Its caller supplies already verified bytes, an explicitly selected clean
absolute parent and one plain basename. Content, provenance, expected
frame count, dimensions and result role remain upstream requirements.
An arbitrary byte slice is not made a verified render by publishing it.

## Operation and ownership

The trusted filesystem supplies CreateTemp, Install and Remove. A
successful create transfers one valid open handle and same-parent name;
failure transfers neither. Successful Install consumes the source name.
A hard-link adapter cannot substitute because its source remains owned.

Before I/O, refuse a missing filesystem, an unclean or nonabsolute parent,
and an empty, dot, dot-dot, slash-containing or NUL-containing basename.
Empty bytes are valid for this byte layer. No root or path is inferred.

Create the private temporary, register cleanup, write the entire body,
and close once before Install. A short write without an error refuses.
Only a successful Install reports confirmed visibility. The zero result
does not mean published and no result grants admission or resume.

The cleanup owner needs only Remove and the temporary handle. Narrow its
private filesystem dependency to that real port so both publishers can
share it. Its existing function body, once-only close, exact owned name
and joined-cause behavior remain unchanged.

## Refusal and uncertainty

Known existing-entry refusal has its own stage, retaining the native
cause. It leaves the incumbent untouched and cleans only this attempt's
private temporary. It never reports the incumbent as a successful result
of this attempt. Reuse needs separate content and provenance verification.

Other installation errors retain an uncertain stage. Cleanup may remove
only the owned private name; it never deletes or rolls back the
destination. A cleanup error joins the primary error without changing
its stage. A failed close is never retried by deferred cleanup.

Success consumes the private name, so deferred cleanup does nothing.
There is no ordinary Rename fallback, destination removal, storage flush,
power-loss guarantee or root protection. Those are separate boundaries.

## Paired witnesses

Generated filesystem and file mocks verify exact arguments and ordered
create, whole-write, close and exclusive-install calls. No Rename method
exists in this port. Unexpected calls, missing calls and premature
publication fail the test. No hand-written canned filesystem is used.

Pair complete payloads with invalid requests; successful creation with
failed creation; whole writes with short and failed writes; successful
close with failed close; first installation with an existing incumbent;
and owned cleanup with uncertain installation plus cleanup failures.

Add arbitrary payload and independent joined-cause Rapid properties.
Existing native exclusion and competing-writer tests remain necessary;
mock success does not establish kernel behavior.

Two tiny real adapter composition tests establish complete first
publication and an untouched
incumbent, including exact namespace entries and bytes.

The new composing body follows actual compile and assertion-red evidence.
Focused red/green checks run during development; independent acceptance
and the complete bar remain required before landing.

At stub 390f3ba, the types and generated mocks compiled with no tests run.
Seventeen of eighteen tests then rejected the empty body through assertions
and missing expected calls. The zero-result anti-test passed. Both exact
Rapid failure records were read and preserved before writing the body.

The composing body now follows the specified order, sharing the existing
cleanup owner. At 1440f86, all 62 publication tests passed, all six Rapid
properties passed 100 cases, and every production statement was covered.
Independent acceptance and the remaining bar gates are still owed.
