# Integration tests

Integration tests check that several components work correctly together.

For the MVP, the API handlers share the same AppData and are connected
to the same HTTP router.

The tests use controlled data so they do not depend on the Internet.

The real API connection is checked separately during the MVP validation
with the running application.
