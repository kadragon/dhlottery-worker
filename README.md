# DHLottery Worker

DHLottery(동행복권) 자동 구매와 알림을 위한 GitHub Actions 기반 서비스입니다.

## 주요 기능

- 🎰 자동 로또 구매 (주 5게임)
- 💰 잔액 모니터링
- 🏆 당첨 확인
- 📱 Telegram 알림
- ⏰ 스케줄 실행 (매주 월요일 10:00 KST)

## 구성

- `agents.md`: 운영 규칙과 개발 원칙
- `plan.md`: 리팩터링/개선 계획과 현황
- `src/`: 실행 코드
- `.github/workflows/lottery.yml`: GitHub Actions 워크플로

## 설치

### 사전 준비

- Bun 1.x
- GitHub 계정 (GitHub Actions 사용)
- DHLottery 계정
- Telegram 봇

### 설치 방법

1. 저장소 클론:
```bash
git clone <repository-url>
cd dhlottery-worker
```

2. 의존성 설치:
```bash
bun install
```

3. GitHub Secrets 설정:
   - 저장소 Settings → Secrets and variables → Actions
   - 다음 항목 추가:
     - `USER_ID`: DHLottery 사용자 ID
     - `PASSWORD`: DHLottery 비밀번호
     - `TELEGRAM_BOT_TOKEN`: Telegram 봇 토큰
     - `TELEGRAM_CHAT_ID`: Telegram 채팅 ID

## 개발

로컬 실행 (.env 파일 필요):
```bash
bun run start
```

테스트 실행:
```bash
bun run test
```

커버리지 실행:
```bash
bun run test:coverage
```

CI 커버리지 게이트 실행:
```bash
bun run test:coverage:ci
```

테스트 감시 모드:
```bash
bun run test:watch
```

타입 체크:
```bash
bun run typecheck
```

## 배포

GitHub Actions로 자동 실행됩니다.

- **스케줄**: 매주 월요일 10:00 KST (01:00 UTC)
- **워크플로**: `.github/workflows/lottery.yml`
- **수동 실행**: GitHub Actions 탭에서 “Run workflow” 사용

## 실행 흐름

1. **세션 초기화** → GET `/login` (DHJSESSIONID 쿠키 발급)
2. **RSA 키 조회** → GET `/login/selectRsaModulus.do`
3. **인증** → POST `/login/securityLoginCheck.do` (RSA 암호화 로그인)
4. **계정 조회** → 잔액 `/mypage/selectUserMndp.do`, 회차 `/lt645/selectThsLt645Info.do`
5. **잔액 확인** → 최소 잔액(5,000원) 확인
6. **구매** → `/olotto/game/egovUserReadySocket.json` → `/olotto/game/execBuy.do`
7. **당첨 확인** → `/myPage.do?method=lottoBuyList`
8. **알림** → Telegram 메시지 전송

## 비즈니스 규칙

- **최소 잔액**: 5,000원 (5게임 × 1,000원)
- **구매 방식**: 자동 번호 생성 (genType "0")
- **당첨 확인 범위**: 이전 주(월~일, KST), 1등만 알림
- **실행 시간**: 매주 월요일 10:00 KST

## 보안

- 모든 인증 정보는 GitHub Secrets로 관리
- 로그/코드에 민감 정보 저장 금지
- 상태 없는 실행 (영구 저장 없음)
- 개인 용도 전제

## 테스트

이 프로젝트는 **TDD(Test-Driven Development)**를 따릅니다.

1. 테스트부터 작성 (RED)
2. 최소 구현으로 통과 (GREEN)
3. 구조 개선 (REFACTOR)

### 테스트 커버리지

- 커버리지 임계치(글로벌):
  - Statements: **85%**
  - Branches: **75%**
  - Functions: **85%**
  - Lines: **85%**
- DHLottery 실 HTML 기반 픽스처
- HTTP 클라이언트 모킹으로 결정적 테스트

#### 커버리지 트러블슈팅

- `Cannot find dependency '@vitest/coverage-v8'` 오류가 발생하면 의존성을 다시 설치하세요:
  ```bash
  bun install
  ```

## 유지보수

### 기능 추가 절차

1. `plan.md`에 작업 목적/범위 정리
2. 테스트 작성 후 구현 (RED → GREEN → REFACTOR)
3. 변경 사항을 `agents.md` 규칙에 맞게 정리

### 디버깅

GitHub Actions 로그 확인:
- Actions 탭 → 최신 워크플로 → “Run lottery workflow” 단계

로컬 테스트 상세 로그:
```bash
bun run test -- --reporter=verbose
```

디버그 모드:
```bash
DEBUG=true bun run start
```

## 구현 메모

### 인증 흐름 (2026-01 기준)
- 세션 초기화 후 RSA 키 조회 및 암호화 로그인 수행
- 로그인 응답이 3xx일 수 있으며, `loginSuccess.do` 포함 여부로 성공 판단
- 리다이렉트는 수동 처리

### 계정 정보
- 잔액: `/mypage/selectUserMndp.do` JSON API (`crntEntrsAmt`)
- 회차: `/lt645/selectThsLt645Info.do` JSON API (`ltEpsd`)

### 구매 프로토콜
- 2단계 원자적 구매 (ready → exec)
- 필요 헤더 및 날짜 파라미터 포함
- 실패 시 전체 흐름 중단 없이 알림 처리

### 당첨 확인
- 이전 주 구매 내역만 조회, 1등만 알림
- 파싱 실패는 빈 결과로 종료

## 참고

- [roeniss/dhlottery-api](https://github.com/roeniss/dhlottery-api)

## 면책

개인 자동화 도구이며 상업적 사용을 금지합니다.
