# Regression tests

Regression tests make sure that a feature that already worked
is not broken by a future change.

Current regression checks:

- the home route returns HTTP 200;
- the home page contains the Groupie Tracker title;
- an unknown route returns HTTP 404;
- the server port keeps its expected behavior.

Some regression tests are currently located in `main_test.go` because
they need access to functions from the `main` package.

## HTTP 403

HTTP 403 behavior belongs to another project ticket.

It is not implemented by this MVP checkpoint, so we do not create
a fake 403 test here.

When the 403 feature is implemented, its regression test can be added.
