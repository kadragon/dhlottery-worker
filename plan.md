# Plan: Account balance API migration (JS-rendered page fix)

## Context
- `/mypage/home` is JS-rendered, HTML parsing fails
- New API: `/mypage/selectUserMndp.do` returns JSON with `crntEntrsAmt` (available balance)

## Tests
- [x] TEST-ACCOUNT-008: parses balance from selectUserMndp.do JSON API response
- [x] TEST-ACCOUNT-009: throws error when API returns non-200 status (covered in TEST-ACCOUNT-001)
- [x] TEST-ACCOUNT-010: throws error when crntEntrsAmt is missing from response (covered in TEST-ACCOUNT-002)
