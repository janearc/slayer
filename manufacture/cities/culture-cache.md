# Embedded culture loading

A city construction starts with a checked set of building elements merged
from the common, parent and named YAML files. Repeated construction of
the same named culture reuses that checked data. Only names of embedded
files can occupy the cache. Load and validate each named culture on first
use, including when several constructors ask for it concurrently.

Drawers still receive independent data, including the options and their
colours. The cache holds no seed, draw state or geometry. Common data is
still not a selectable culture, and unknown names still report the list
of available cultures. Use the existing geometry digests to check that
construction output stays identical.
