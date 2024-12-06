Technical Debt
==============

This is a list of all technical debt I have taken on, named, and with an explanation, so I have some way of tracking it.

Each item will be placed at the nearest location in code and be called out as: `@techdebt(N): any explainer`.

## Listing

1. Setup all [responseHandling][htmx-response-handling] to be `swap: true` which is likely too broad, because I didn't want to figure out how to handle error conditions or debug them when tests were failing. By swapping I at least got the error to show up on the page.

[htmx-response-handling]: https://htmx.org/docs/#response-handling
