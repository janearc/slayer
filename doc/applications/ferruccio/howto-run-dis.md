# howto-run-dis

ferruccio is an ordinary mac app. it is not signed with an apple developer
key, so macos asks you to say that you mean to open it, once. this page is
the whole of it.

you need a mac with apple silicon (m1 or later). the app is not built for
intel macs.

## install

1. open `Ferruccio.dmg`.
2. drag `Ferruccio` onto `Applications`.

## open it the first time

1. open `Ferruccio` from `Applications`. macos will say it cannot verify
   the app and will not open it. close that message. do not move the app
   to the trash.
2. open `System Settings`, then `Privacy & Security`.
3. scroll to `Security`. it says `Ferruccio` was blocked. click
   `Open Anyway`, enter your password, and click `Open`.

macos remembers this. from then on `Ferruccio` opens like any other app.

if you prefer the terminal, this clears the same flag:

```sh
xattr -dr com.apple.quarantine /Applications/Ferruccio.app
```

## check what you downloaded

the release has a `Ferruccio.dmg.sha256` file beside the image. these two
should print the same string:

```sh
shasum -a 256 Ferruccio.dmg
cat Ferruccio.dmg.sha256
```

## remove it

drag `Ferruccio` from `Applications` to the trash.
