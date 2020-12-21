# Go tools

Various packages and cli tools of things I continuously need in my Go projects.

## Packages

### sqlbuilder

An simple SQL statement builder that is more about allowing other packages to
build on top of it. It has a good set of functions and options to generate a
pretty good range of select queries, but if you need to, you can wrap the
smaller primitives and create much more complex queries. Most of the package is
exported so you can take statement objects and add to them with `StatementOption`s
