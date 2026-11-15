# testdata

The database files the metals' optics are read from, committed whole and
unchanged, under the paths they have inside the release zip.

They come from the refractiveindex.info database release v2026-05-24,
whose archive is rii-database-2026-05-24.zip, 13440092 bytes, sha256
6cdebebd17fe02b6dc2b03189e15e6c689afc6619c558e1df7b9ac307b6d14eb,
fetched from
https://github.com/polyanskiy/refractiveindex.info-database/releases/download/v2026-05-24/rii-database-2026-05-24.zip
on 2026-11-11. Each file's own header states the database's licence, in
its words: "refractiveindex.info database is in the public domain" and
"copyright and related rights waived via CC0 1.0". Cite the database as
Polyanskiy, M. N. Refractiveindex.info database of optical constants,
Scientific Data 11, 94 (2024), doi 10.1038/s41597-023-02898-2.

The files, and the paper each row the package reads comes from:

- main/Au/nk/Johnson.yml, gold as P. B. Johnson and R. W. Christy
  measured it: Optical constants of the noble metals, Phys. Rev. B 6,
  4370-4379 (1972).
- main/Ag/nk/Johnson.yml, silver as Johnson and Christy measured it, the
  same paper.
- main/Ti/nk/Johnson.yml, titanium as Johnson and Christy measured it:
  Optical constants of transition metals: Ti, V, Cr, Mn, Fe, Co, Ni, and
  Pd, Phys. Rev. B 9, 5056-5070 (1974).
- main/Al/nk/Rakic-BB.yml, aluminium as A. D. Rakić, A. B. Djurišic,
  J. M. Elazar and M. L. Majewski fit it: Optical properties of metallic
  films for vertical-cavity optoelectronic devices, Appl. Opt. 37,
  5271-5283 (1998), the Brendel-Bormann model.
- main/Be/nk/Rakic-LD.yml, beryllium by the same paper's
  Lorentz-Drude model.

Aluminium's blue n was corrected to the published row on 2026-11-11, by
decision: scene/thing/metals.go had 0.6444 for it where the
published row, line 63 of main/Al/nk/Rakic-BB.yml, has 6.4439e-01, so
thing now carries the published value. All five metals' values match
their rows and thing/metals.go exactly; the tests read the rows back
from these files to prove it.

The archive was fetched and these files taken from it, byte for byte:

    curl -L -o rii-database-2026-05-24.zip \
      https://github.com/polyanskiy/refractiveindex.info-database/releases/download/v2026-05-24/rii-database-2026-05-24.zip
    unzip rii-database-2026-05-24.zip \
      'database/data/main/Au/nk/Johnson.yml' \
      'database/data/main/Ag/nk/Johnson.yml' \
      'database/data/main/Ti/nk/Johnson.yml' \
      'database/data/main/Al/nk/Rakic-BB.yml' \
      'database/data/main/Be/nk/Rakic-LD.yml'
    cp database/data/main/Au/nk/Johnson.yml  testdata/main/Au/nk/Johnson.yml
    cp database/data/main/Ag/nk/Johnson.yml  testdata/main/Ag/nk/Johnson.yml
    cp database/data/main/Ti/nk/Johnson.yml  testdata/main/Ti/nk/Johnson.yml
    cp database/data/main/Al/nk/Rakic-BB.yml testdata/main/Al/nk/Rakic-BB.yml
    cp database/data/main/Be/nk/Rakic-LD.yml testdata/main/Be/nk/Rakic-LD.yml

Each copy was compared with cmp against the unzipped original before
committing.
