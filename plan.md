# Plan: Login redirect success regression (manual redirects)

## Tests
- [x] TEST-AUTH-005: treats 302 login redirect as success when redirects are manual and body is empty
- [x] TEST-AUTH-006: rejects 302 redirect when Location does not point to login success
